package com.att.rif.extractor.aop;

import com.att.rif.extractor.common.NodeIdComputer;
import com.att.rif.extractor.common.SourceRefBuilder;
import com.github.javaparser.ParserConfiguration;
import com.github.javaparser.StaticJavaParser;
import com.github.javaparser.ast.CompilationUnit;
import com.github.javaparser.ast.body.ClassOrInterfaceDeclaration;
import com.github.javaparser.ast.body.MethodDeclaration;
import com.github.javaparser.ast.expr.AnnotationExpr;
import com.github.javaparser.ast.expr.NormalAnnotationExpr;
import com.github.javaparser.ast.expr.SingleMemberAnnotationExpr;
import com.github.javaparser.ast.expr.StringLiteralExpr;
import com.github.javaparser.ast.visitor.VoidVisitorAdapter;
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
 * Scans Java source files for Spring AOP constructs and emits Tier-C {@code ADVISES} edges.
 *
 * <h3>Detection strategy</h3>
 * <ol>
 *   <li>Identify classes annotated with {@code @Aspect}.</li>
 *   <li>Within each aspect class, find methods annotated with {@code @Around}, {@code @Before},
 *       {@code @After}, {@code @AfterReturning}, or {@code @AfterThrowing}.</li>
 *   <li>Extract the pointcut expression string from the annotation value.</li>
 *   <li>Emit one {@code ADVISES} edge from the aspect CLASS node to a synthetic
 *       {@code POINTCUT_EXPRESSION} node per advice method.</li>
 * </ol>
 *
 * <h3>Confidence and caveats</h3>
 * All {@code ADVISES} edges carry {@code confidence=inferred}. The pointcut expression is
 * captured verbatim but not evaluated against the graph — runtime proxy targets (CGLIB/JDK
 * proxies) may differ from the static string match.
 *
 * <h3>Determinism</h3>
 * No wall-clock values or random IDs. All IDs are content-addressed SHA-256 digests.
 */
public class SpringAopExtractor {

    private static final Logger LOG = LoggerFactory.getLogger(SpringAopExtractor.class);

    /** Advice annotations to look for on methods inside @Aspect classes. */
    private static final Set<String> ADVICE_ANNOTATIONS = Set.of(
            "Around", "Before", "After", "AfterReturning", "AfterThrowing"
    );

    static final String ADVISES_CAVEAT =
            "static pointcut match only — runtime proxy targets may differ";

    private final String repoId;
    private final String sha;
    private final Path sourceRoot;

    public SpringAopExtractor(String repoId, String sha, Path sourceRoot) {
        this.repoId = repoId;
        this.sha = sha;
        this.sourceRoot = sourceRoot;
    }

    public record ExtractionResult(List<Map<String, Object>> nodes, List<Map<String, Object>> edges) {
    }

    public ExtractionResult extract() throws IOException {
        List<Map<String, Object>> nodes = new ArrayList<>();
        List<Map<String, Object>> edges = new ArrayList<>();

        CombinedTypeSolver typeSolver = new CombinedTypeSolver(
                new ReflectionTypeSolver(false),
                new JavaParserTypeSolver(sourceRoot)
        );
        ParserConfiguration config = new ParserConfiguration()
                .setSymbolResolver(new JavaSymbolSolver(typeSolver));
        StaticJavaParser.setConfiguration(config);

        try (Stream<Path> stream = Files.walk(sourceRoot)) {
            List<Path> javaFiles = stream
                    .filter(p -> p.toString().endsWith(".java"))
                    .sorted()
                    .toList();

            for (Path file : javaFiles) {
                try {
                    CompilationUnit cu = StaticJavaParser.parse(file);
                    new AopVisitor(repoId, sha, sourceRoot, nodes, edges).visit(cu, null);
                } catch (Exception e) {
                    LOG.warn("Skipping file {} due to parse error: {}", file, e.getMessage());
                }
            }
        }

        return new ExtractionResult(nodes, edges);
    }

    // -----------------------------------------------------------------------
    // Visitor
    // -----------------------------------------------------------------------

    private static class AopVisitor extends VoidVisitorAdapter<Void> {

        private final String repoId;
        private final String sha;
        private final Path sourceRoot;
        private final List<Map<String, Object>> nodes;
        private final List<Map<String, Object>> edges;

        AopVisitor(String repoId, String sha, Path sourceRoot,
                   List<Map<String, Object>> nodes, List<Map<String, Object>> edges) {
            this.repoId = repoId;
            this.sha = sha;
            this.sourceRoot = sourceRoot;
            this.nodes = nodes;
            this.edges = edges;
        }

        @Override
        public void visit(ClassOrInterfaceDeclaration cls, Void arg) {
            // Only process @Aspect classes
            boolean isAspect = cls.getAnnotations().stream()
                    .anyMatch(a -> "Aspect".equals(a.getNameAsString()));
            if (!isAspect) {
                super.visit(cls, arg);
                return;
            }

            String classFqn = resolveClassFqn(cls);
            String classNodeId = NodeIdComputer.computeNodeId(repoId, classFqn, "CLASS");

            // Process each advice method
            for (MethodDeclaration method : cls.getMethods()) {
                for (AnnotationExpr ann : method.getAnnotations()) {
                    String annName = ann.getNameAsString();
                    if (!ADVICE_ANNOTATIONS.contains(annName)) {
                        continue;
                    }

                    String pointcutExpr = extractPointcutExpression(ann);
                    int adviceLine = method.getBegin().map(p -> p.line).orElse(1);

                    // Build synthetic POINTCUT_EXPRESSION node
                    String pointcutNodeId = NodeIdComputer.pointcutExprNodeId(
                            repoId, classFqn, method.getNameAsString(), adviceLine);
                    String pointcutSourceRef = SourceRefBuilder.pointcutExpression(repoId, classFqn, adviceLine);
                    String pointcutLabel = pointcutExpr.length() > 200
                            ? pointcutExpr.substring(0, 200)
                            : pointcutExpr;

                    nodes.add(pointcutExpressionNode(pointcutNodeId, pointcutSourceRef, pointcutLabel));

                    // Build ADVISES edge: aspect CLASS → POINTCUT_EXPRESSION node
                    String edgeSourceRef = buildSourceRef(method);
                    edges.add(advisesEdge(classNodeId, pointcutNodeId, edgeSourceRef,
                            pointcutExpr, toAdviceType(annName)));
                }
            }

            super.visit(cls, arg);
        }

        // ── Node builders ────────────────────────────────────────────────────

        private Map<String, Object> pointcutExpressionNode(String nodeId, String sourceRef, String label) {
            LinkedHashMap<String, Object> m = new LinkedHashMap<>();
            m.put("record_type", "node");
            m.put("node_id", nodeId);
            m.put("repo_id", repoId);
            m.put("kind", "POINTCUT_EXPRESSION");
            m.put("label", label);
            m.put("source_ref", sourceRef);
            m.put("confidence", "inferred");
            m.put("phase_populated", 2);
            m.put("origin", "synthetic");
            m.put("provenance_kind", "stub");
            return m;
        }

        // ── Edge builders ────────────────────────────────────────────────────

        private Map<String, Object> advisesEdge(String fromNodeId, String toNodeId,
                                                 String sourceRef, String pointcutExpr,
                                                 String adviceType) {
            LinkedHashMap<String, Object> m = new LinkedHashMap<>();
            m.put("record_type", "edge");
            m.put("edge_id", NodeIdComputer.computeEdgeId(fromNodeId, "ADVISES", toNodeId));
            m.put("label", "ADVISES");
            m.put("from_node_id", fromNodeId);
            m.put("to_node_id", toNodeId);
            m.put("source_ref", sourceRef);
            m.put("confidence", "inferred");
            m.put("tier", 3);
            m.put("phase_populated", 2);
            m.put("repo_id", repoId);
            m.put("evidence", "pointcut_expression");
            m.put("pointcut_expr", pointcutExpr);
            m.put("advice_type", adviceType);
            m.put("completeness_caveat", ADVISES_CAVEAT);
            return m;
        }

        // ── Helper methods ───────────────────────────────────────────────────

        /**
         * Extracts the pointcut expression string from an advice annotation.
         *
         * <p>Handles three annotation forms:
         * <ul>
         *   <li>{@code @Around("execution(* com.example.*.*(..))")} — SingleMemberAnnotation</li>
         *   <li>{@code @Around(value = "execution(* com.example.*.*(..))")} — NormalAnnotation</li>
         * </ul>
         */
        private static String extractPointcutExpression(AnnotationExpr ann) {
            if (ann instanceof SingleMemberAnnotationExpr single) {
                if (single.getMemberValue() instanceof StringLiteralExpr str) {
                    return str.asString();
                }
                return single.getMemberValue().toString();
            }
            if (ann instanceof NormalAnnotationExpr normal) {
                return normal.getPairs().stream()
                        .filter(p -> "value".equals(p.getNameAsString()) || "pointcut".equals(p.getNameAsString()))
                        .findFirst()
                        .map(p -> p.getValue() instanceof StringLiteralExpr s ? s.asString() : p.getValue().toString())
                        .orElse("");
            }
            return "";
        }

        private String resolveClassFqn(ClassOrInterfaceDeclaration cls) {
            try {
                return cls.resolve().getQualifiedName();
            } catch (Exception e) {
                Optional<CompilationUnit> cu = cls.findCompilationUnit();
                String pkg = cu.flatMap(c -> c.getPackageDeclaration())
                        .map(pd -> pd.getNameAsString())
                        .orElse("");
                return pkg.isEmpty() ? cls.getNameAsString() : pkg + "." + cls.getNameAsString();
            }
        }

        private String buildSourceRef(MethodDeclaration method) {
            Optional<Path> cuPath = method.findCompilationUnit()
                    .flatMap(cu -> cu.getStorage())
                    .map(s -> s.getPath());
            int line = method.getBegin().map(p -> p.line).orElse(1);
            if (cuPath.isPresent()) {
                String rel = sourceRoot.relativize(cuPath.get()).toString().replace('\\', '/');
                return SourceRefBuilder.build(repoId, sha, rel, line);
            }
            return SourceRefBuilder.build(repoId, sha, "unknown", line);
        }

        private static String toAdviceType(String annName) {
            return switch (annName) {
                case "Around"         -> "around";
                case "Before"         -> "before";
                case "After"          -> "after";
                case "AfterReturning" -> "afterReturning";
                case "AfterThrowing"  -> "afterThrowing";
                default               -> annName.toLowerCase();
            };
        }
    }
}
