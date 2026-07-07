package com.att.rif.extractor.di;

import com.att.rif.extractor.common.LombokUtil;
import com.att.rif.extractor.common.NodeIdComputer;
import com.att.rif.extractor.common.SourceRefBuilder;
import com.github.javaparser.ParserConfiguration;
import com.github.javaparser.StaticJavaParser;
import com.github.javaparser.ast.CompilationUnit;
import com.github.javaparser.ast.Node;
import com.github.javaparser.ast.body.ClassOrInterfaceDeclaration;
import com.github.javaparser.ast.body.ConstructorDeclaration;
import com.github.javaparser.ast.body.FieldDeclaration;
import com.github.javaparser.ast.body.MethodDeclaration;
import com.github.javaparser.ast.body.Parameter;
import com.github.javaparser.ast.body.VariableDeclarator;
import com.github.javaparser.ast.expr.AnnotationExpr;
import com.github.javaparser.ast.visitor.VoidVisitorAdapter;
import com.github.javaparser.resolution.UnsolvedSymbolException;
import com.github.javaparser.symbolsolver.JavaSymbolSolver;
import com.github.javaparser.symbolsolver.resolution.typesolvers.CombinedTypeSolver;
import com.github.javaparser.symbolsolver.resolution.typesolvers.JavaParserTypeSolver;
import com.github.javaparser.symbolsolver.resolution.typesolvers.ReflectionTypeSolver;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.Set;
import java.util.stream.Stream;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * Scans Java source files under a source root for Spring DI annotations and emits
 * Tier-B graph edges: INJECTS, PRODUCES, REGISTERS.
 *
 * <h3>Edge semantics</h3>
 * <ul>
 *   <li>{@code INJECTS}: RECEIVER_CLASS → INJECTED_TYPE_CLASS.
 *       Triggered by {@code @Autowired} or {@code @Inject} on a field, constructor, or setter.</li>
 *   <li>{@code PRODUCES}: METHOD → RETURN_TYPE_CLASS.
 *       Triggered by {@code @Bean} on a method inside a {@code @Configuration} class.</li>
 *   <li>{@code REGISTERS}: CLASS → APPLICATION_CONTEXT.
 *       Triggered by any Spring stereotype annotation on a class declaration.</li>
 * </ul>
 *
 * <h3>Determinism</h3>
 * No wall-clock values, UUIDs, or unordered collections are used in ID computation.
 * All node and edge IDs are content-addressed SHA-256 digests. Ordering is applied
 * in {@link com.att.rif.extractor.common.EmitHelper} before writing output.
 */
public class SpringDiExtractor {

    private static final Logger LOG = LoggerFactory.getLogger(SpringDiExtractor.class);

    // Stereotype annotations that trigger REGISTERS edges
    private static final Set<String> STEREOTYPE_ANNOTATIONS = Set.of(
            "Component", "Service", "Repository",
            "Controller", "RestController", "Configuration"
    );

    // Injection annotations that trigger INJECTS edges
    private static final Set<String> INJECT_ANNOTATIONS = Set.of("Autowired", "Inject");

    private static final String INJECTS_CAVEAT =
            "Misses programmatic ApplicationContext.getBean() lookups. Qualifier ambiguity (@Qualifier) " +
            "not resolved — one INJECTS edge per injection point regardless of which bean is selected at " +
            "runtime. @Primary, @ConditionalOn*, and BeanDefinitionRegistry-based programmatic registration " +
            "not captured.";

    private static final String PRODUCES_CAVEAT =
            "Conditional beans (@ConditionalOnProperty, @Profile) treated as always-present — extractor " +
            "does not evaluate conditions. Generic/parameterized return types may not resolve fully via " +
            "SymbolSolver. Programmatic bean registration via BeanDefinitionRegistry is not visible.";

    private static final String REGISTERS_CAVEAT =
            "XML-based bean registration not covered. @Import and @ImportResource not covered. " +
            "Meta-annotations (stereotypes composed inside a custom annotation) are not resolved " +
            "unless the annotation itself is in the stereotype list.";

    private final String repoId;
    private final String sha;
    private final Path sourceRoot;

    public SpringDiExtractor(String repoId, String sha, Path sourceRoot) {
        this.repoId = repoId;
        this.sha = sha;
        this.sourceRoot = sourceRoot;
    }

    public record ExtractionResult(List<Map<String, Object>> nodes, List<Map<String, Object>> edges) {
    }

    public ExtractionResult extract() throws IOException {
        List<Map<String, Object>> nodes = new ArrayList<>();
        List<Map<String, Object>> edges = new ArrayList<>();

        // Configure JavaParser with SymbolSolver for type resolution
        CombinedTypeSolver typeSolver = new CombinedTypeSolver(
                new ReflectionTypeSolver(false),
                new JavaParserTypeSolver(sourceRoot)
        );
        ParserConfiguration config = new ParserConfiguration()
                .setSymbolResolver(new JavaSymbolSolver(typeSolver));
        StaticJavaParser.setConfiguration(config);

        // Emit the APPLICATION_CONTEXT synthetic node (once per run)
        nodes.add(applicationContextNode());

        // Pre-compute APPLICATION_CONTEXT node_id
        String appCtxNodeId = NodeIdComputer.applicationContextNodeId(repoId);

        // Walk all .java files under sourceRoot
        try (Stream<Path> stream = Files.walk(sourceRoot)) {
            List<Path> javaFiles = stream
                    .filter(p -> p.toString().endsWith(".java"))
                    .sorted()   // deterministic traversal order
                    .toList();

            for (Path file : javaFiles) {
                try {
                    CompilationUnit cu = StaticJavaParser.parse(file);
                    new DiVisitor(repoId, sha, sourceRoot, appCtxNodeId, edges)
                            .visit(cu, null);
                } catch (Exception e) {
                    LOG.warn("Skipping file {} due to parse error: {}", file, e.getMessage());
                }
            }
        }

        return new ExtractionResult(nodes, edges);
    }

    // -----------------------------------------------------------------------
    // Synthetic node builders
    // -----------------------------------------------------------------------

    private Map<String, Object> applicationContextNode() {
        LinkedHashMap<String, Object> m = new LinkedHashMap<>();
        m.put("record_type", "node");
        m.put("node_id", NodeIdComputer.applicationContextNodeId(repoId));
        m.put("repo_id", repoId);
        m.put("qualified_name", "APPLICATION_CONTEXT:" + repoId);
        m.put("kind", "CLASS");
        m.put("source_ref", SourceRefBuilder.applicationContext(repoId));
        m.put("confidence", "probable");
        m.put("phase_populated", 2);
        m.put("origin", "virtual");
        m.put("provenance_kind", "stub");
        m.put("label", "ApplicationContext");
        return m;
    }

    // -----------------------------------------------------------------------
    // Visitor
    // -----------------------------------------------------------------------

    private static class DiVisitor extends VoidVisitorAdapter<Void> {

        private final String repoId;
        private final String sha;
        private final Path sourceRoot;
        private final String appCtxNodeId;
        private final List<Map<String, Object>> edges;

        DiVisitor(String repoId, String sha, Path sourceRoot,
                  String appCtxNodeId, List<Map<String, Object>> edges) {
            this.repoId = repoId;
            this.sha = sha;
            this.sourceRoot = sourceRoot;
            this.appCtxNodeId = appCtxNodeId;
            this.edges = edges;
        }

        @Override
        public void visit(ClassOrInterfaceDeclaration cls, Void arg) {
            if (cls.isInterface()) {
                super.visit(cls, arg);
                return;
            }

            String classFqn = resolveClassFqn(cls);
            String classNodeId = NodeIdComputer.computeNodeId(repoId, classFqn, "CLASS");
            String classSourceRef = buildSourceRef(cls);

            // ── REGISTERS edges ──────────────────────────────────────────────
            for (AnnotationExpr ann : cls.getAnnotations()) {
                String annName = ann.getNameAsString();
                if (STEREOTYPE_ANNOTATIONS.contains(annName)) {
                    String stereotype = toStereotype(annName);
                    edges.add(registersEdge(classNodeId, appCtxNodeId, classSourceRef, annName, stereotype));
                    break; // one REGISTERS edge per class (first matching stereotype)
                }
            }

            // ── INJECTS edges from @Autowired / @Inject fields ────────────────
            for (FieldDeclaration field : cls.getFields()) {
                // G7: Skip Lombok-generated fields (e.g., @Slf4j log field)
                if (LombokUtil.isLombokGeneratedField(cls, field)) {
                    continue;
                }
                if (!hasInjectAnnotation(field.getAnnotations())) {
                    continue;
                }
                String injectedTypeFqn = resolveTypeFqn(field.getElementType().asString(), field);
                String injectedNodeId = NodeIdComputer.computeNodeId(repoId, injectedTypeFqn, "CLASS");
                String fieldSourceRef = buildSourceRef(field);
                String annotationName = firstInjectAnnotationName(field.getAnnotations());
                edges.add(injectsEdge(classNodeId, injectedNodeId, fieldSourceRef,
                        annotationName, "field", injectedTypeFqn, null));
            }

            // ── INJECTS edges from @Autowired constructors ────────────────────
            for (ConstructorDeclaration ctor : cls.getConstructors()) {
                if (!hasInjectAnnotation(ctor.getAnnotations())) {
                    continue;
                }
                String ctorSourceRef = buildSourceRef(ctor);
                String annotationName = firstInjectAnnotationName(ctor.getAnnotations());
                for (Parameter param : ctor.getParameters()) {
                    String injectedTypeFqn = resolveTypeFqn(param.getType().asString(), param);
                    String injectedNodeId = NodeIdComputer.computeNodeId(repoId, injectedTypeFqn, "CLASS");
                    edges.add(injectsEdge(classNodeId, injectedNodeId, ctorSourceRef,
                            annotationName, "constructor", injectedTypeFqn, null));
                }
            }

            // ── INJECTS edges from @Autowired setter methods ──────────────────
            // ── PRODUCES edges from @Bean methods ────────────────────────────
            for (MethodDeclaration method : cls.getMethods()) {
                List<AnnotationExpr> methodAnns = method.getAnnotations();

                // @Autowired setter
                if (hasInjectAnnotation(methodAnns)) {
                    String methodSourceRef = buildSourceRef(method);
                    String annotationName = firstInjectAnnotationName(methodAnns);
                    for (Parameter param : method.getParameters()) {
                        String injectedTypeFqn = resolveTypeFqn(param.getType().asString(), param);
                        String injectedNodeId = NodeIdComputer.computeNodeId(repoId, injectedTypeFqn, "CLASS");
                        edges.add(injectsEdge(classNodeId, injectedNodeId, methodSourceRef,
                                annotationName, "setter", injectedTypeFqn, null));
                    }
                }

                // @Bean factory method
                if (hasAnnotation(methodAnns, "Bean")) {
                    String methodFqn = buildMethodFqn(cls, method);
                    String methodNodeId = NodeIdComputer.computeNodeId(repoId, methodFqn, "METHOD");
                    String methodSourceRef = buildSourceRef(method);
                    String returnTypeFqn = resolveTypeFqn(method.getType().asString(), method);
                    String returnTypeNodeId = NodeIdComputer.computeNodeId(repoId, returnTypeFqn, "CLASS");
                    edges.add(producesEdge(methodNodeId, returnTypeNodeId, methodSourceRef, returnTypeFqn));
                }
            }

            super.visit(cls, arg);
        }

        // ── Edge builders ────────────────────────────────────────────────────

        private Map<String, Object> registersEdge(String fromNodeId, String toNodeId,
                                                   String sourceRef, String annName,
                                                   String stereotype) {
            LinkedHashMap<String, Object> m = edgeBase("REGISTERS", fromNodeId, toNodeId, sourceRef);
            m.put("confidence", "probable");
            m.put("tier", 2);
            m.put("evidence", "@" + annName);
            m.put("stereotype", stereotype);
            m.put("completeness_caveat", REGISTERS_CAVEAT);
            return m;
        }

        private Map<String, Object> injectsEdge(String fromNodeId, String toNodeId,
                                                 String sourceRef, String annName,
                                                 String injectionType, String declaredType,
                                                 String qualifier) {
            LinkedHashMap<String, Object> m = edgeBase("INJECTS", fromNodeId, toNodeId, sourceRef);
            m.put("confidence", "probable");
            m.put("tier", 2);
            m.put("evidence", "@" + annName);
            m.put("injection_type", injectionType);
            m.put("declared_type", declaredType);
            m.put("qualifier", qualifier);
            m.put("completeness_caveat", INJECTS_CAVEAT);
            return m;
        }

        private Map<String, Object> producesEdge(String fromNodeId, String toNodeId,
                                                  String sourceRef, String returnTypeFqn) {
            LinkedHashMap<String, Object> m = edgeBase("PRODUCES", fromNodeId, toNodeId, sourceRef);
            m.put("confidence", "probable");
            m.put("tier", 2);
            m.put("evidence", "@Bean");
            m.put("produced_type", returnTypeFqn);
            m.put("completeness_caveat", PRODUCES_CAVEAT);
            return m;
        }

        private LinkedHashMap<String, Object> edgeBase(String label, String fromNodeId,
                                                        String toNodeId, String sourceRef) {
            LinkedHashMap<String, Object> m = new LinkedHashMap<>();
            m.put("record_type", "edge");
            m.put("edge_id", NodeIdComputer.computeEdgeId(fromNodeId, label, toNodeId));
            m.put("label", label);
            m.put("from_node_id", fromNodeId);
            m.put("to_node_id", toNodeId);
            m.put("source_ref", sourceRef);
            m.put("phase_populated", 2);
            m.put("repo_id", repoId);
            return m;
        }

        // ── Helper methods ───────────────────────────────────────────────────

        private String resolveClassFqn(ClassOrInterfaceDeclaration cls) {
            try {
                return cls.resolve().getQualifiedName();
            } catch (Exception e) {
                // Fall back: package + simple name
                Optional<CompilationUnit> cu = cls.findCompilationUnit();
                String pkg = cu.flatMap(c -> c.getPackageDeclaration())
                        .map(pd -> pd.getNameAsString())
                        .orElse("");
                return pkg.isEmpty() ? cls.getNameAsString() : pkg + "." + cls.getNameAsString();
            }
        }

        private String resolveTypeFqn(String simpleOrFqn, Node contextNode) {
            // Try SymbolSolver-backed resolution on the AST node that declares the type
            try {
                if (contextNode instanceof FieldDeclaration fd) {
                    return fd.resolve().getType().describe();
                }
                if (contextNode instanceof Parameter param) {
                    return param.resolve().getType().describe();
                }
                if (contextNode instanceof MethodDeclaration md) {
                    return md.resolve().getReturnType().describe();
                }
            } catch (UnsolvedSymbolException | UnsupportedOperationException e) {
                LOG.debug("Cannot resolve type '{}': {}", simpleOrFqn, e.getMessage());
            } catch (Exception e) {
                LOG.debug("Unexpected error resolving type '{}': {}", simpleOrFqn, e.getMessage());
            }
            // Fall back: use simple name with "?" suffix to flag unresolved
            return simpleOrFqn.contains(".") ? simpleOrFqn : simpleOrFqn + "?";
        }

        private String buildMethodFqn(ClassOrInterfaceDeclaration cls, MethodDeclaration method) {
            String classFqn = resolveClassFqn(cls);
            StringBuilder sb = new StringBuilder(classFqn).append('#').append(method.getNameAsString()).append('(');
            List<Parameter> params = method.getParameters();
            for (int i = 0; i < params.size(); i++) {
                if (i > 0) sb.append(',');
                sb.append(resolveTypeFqn(params.get(i).getType().asString(), params.get(i)));
            }
            sb.append(')');
            return sb.toString();
        }

        private String buildSourceRef(Node node) {
            Optional<Path> cuPath = node.findCompilationUnit()
                    .flatMap(cu -> cu.getStorage())
                    .map(s -> s.getPath());
            int line = node.getBegin().map(p -> p.line).orElse(1);
            if (cuPath.isPresent()) {
                String rel = sourceRoot.relativize(cuPath.get()).toString().replace('\\', '/');
                return SourceRefBuilder.build(repoId, sha, rel, line);
            }
            return SourceRefBuilder.build(repoId, sha, "unknown", line);
        }

        private boolean hasInjectAnnotation(List<AnnotationExpr> annotations) {
            return annotations.stream()
                    .anyMatch(a -> INJECT_ANNOTATIONS.contains(a.getNameAsString()));
        }

        private boolean hasAnnotation(List<AnnotationExpr> annotations, String name) {
            return annotations.stream().anyMatch(a -> a.getNameAsString().equals(name));
        }

        private String firstInjectAnnotationName(List<AnnotationExpr> annotations) {
            return annotations.stream()
                    .filter(a -> INJECT_ANNOTATIONS.contains(a.getNameAsString()))
                    .map(AnnotationExpr::getNameAsString)
                    .findFirst()
                    .orElse("Autowired");
        }

        private static String toStereotype(String annName) {
            return switch (annName) {
                case "Service" -> "service";
                case "Repository" -> "repository";
                case "Controller", "RestController" -> "controller";
                case "Configuration" -> "configuration";
                default -> "component";
            };
        }
    }
}
