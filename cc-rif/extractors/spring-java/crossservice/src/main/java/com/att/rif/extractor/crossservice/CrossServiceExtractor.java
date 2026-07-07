package com.att.rif.extractor.crossservice;

import com.att.rif.extractor.common.NodeIdComputer;
import com.att.rif.extractor.common.SourceRefBuilder;
import com.github.javaparser.ParserConfiguration;
import com.github.javaparser.StaticJavaParser;
import com.github.javaparser.ast.CompilationUnit;
import com.github.javaparser.ast.body.ClassOrInterfaceDeclaration;
import com.github.javaparser.ast.body.MethodDeclaration;
import com.github.javaparser.ast.expr.AnnotationExpr;
import com.github.javaparser.ast.expr.MethodCallExpr;
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
import java.util.HashSet;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.Set;
import java.util.stream.Stream;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

/**
 * Scans Java source files for cross-service call patterns and emits Tier-C edges:
 * {@code CALLS_SOAP} and {@code CALLS_REST}.
 *
 * <h3>CALLS_SOAP detection</h3>
 * Finds classes annotated with {@code @WebServiceClient} (JAX-WS generated stubs).
 * For methods in OTHER classes that hold or use a field of the stub type, emits a
 * {@code CALLS_SOAP} edge from the containing method to the stub class node.
 *
 * <h3>CALLS_REST detection</h3>
 * <ul>
 *   <li>{@code @FeignClient} interfaces — emits {@code CALLS_REST} from any method that
 *       has a field of the Feign client type to the Feign interface node.</li>
 *   <li>{@code RestTemplate} call sites — detects calls to {@code getForObject},
 *       {@code postForObject}, {@code exchange}, {@code getForEntity}, {@code postForEntity},
 *       {@code put}, {@code delete} on RestTemplate instances; emits to a synthetic
 *       {@code URL_ENDPOINT} node.</li>
 *   <li>{@code WebClient} call sites — detects {@code get()}, {@code post()}, {@code put()},
 *       {@code delete()}, {@code patch()} on WebClient instances.</li>
 * </ul>
 *
 * <h3>Confidence and caveats</h3>
 * All edges carry {@code confidence=inferred}. URL strings are not resolved at extraction
 * time. Dynamic port creation ({@code Service.getPort()} with runtime URL) is not covered.
 *
 * <h3>Determinism</h3>
 * No wall-clock values or random IDs. All IDs are content-addressed SHA-256 digests.
 */
public class CrossServiceExtractor {

    private static final Logger LOG = LoggerFactory.getLogger(CrossServiceExtractor.class);

    private static final Set<String> REST_TEMPLATE_METHODS = Set.of(
            "getForObject", "getForEntity", "postForObject", "postForEntity",
            "exchange", "put", "delete", "patchForObject", "execute"
    );

    private static final Set<String> WEB_CLIENT_METHODS = Set.of(
            "get", "post", "put", "delete", "patch", "head", "options"
    );

    static final String CALLS_SOAP_CAVEAT =
            "Target endpoint URL not resolved — only stub class usage tracked. " +
            "Dynamic port creation (Service.getPort() with runtime-provided URL) not covered. " +
            "Dynamic WSDL URLs constructed at runtime are not capturable.";

    static final String CALLS_REST_CAVEAT =
            "URL strings are not resolved at extraction time — target is symbolic. " +
            "HTTP method not always determinable from RestTemplate usage patterns. " +
            "URI path may be partially dynamic. Calls through gateway abstractions not captured.";

    private final String repoId;
    private final String sha;
    private final Path sourceRoot;

    public CrossServiceExtractor(String repoId, String sha, Path sourceRoot) {
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

        // First pass: collect SOAP stub class FQNs and FeignClient interface FQNs
        Set<String> soapStubFqns = new HashSet<>();
        Set<String> feignClientFqns = new HashSet<>();
        List<Path> javaFiles;
        try (Stream<Path> stream = Files.walk(sourceRoot)) {
            javaFiles = stream.filter(p -> p.toString().endsWith(".java")).sorted().toList();
        }

        for (Path file : javaFiles) {
            try {
                CompilationUnit cu = StaticJavaParser.parse(file);
                new StubCollectorVisitor(soapStubFqns, feignClientFqns).visit(cu, null);
            } catch (Exception e) {
                LOG.debug("First-pass skipping {}: {}", file, e.getMessage());
            }
        }

        // Emit FeignClient interface nodes
        for (String fqn : feignClientFqns) {
            nodes.add(feignClientNode(fqn));
        }

        // Second pass: find call sites
        for (Path file : javaFiles) {
            try {
                CompilationUnit cu = StaticJavaParser.parse(file);
                new CallSiteVisitor(repoId, sha, sourceRoot, soapStubFqns, feignClientFqns,
                        nodes, edges).visit(cu, null);
            } catch (Exception e) {
                LOG.debug("Second-pass skipping {}: {}", file, e.getMessage());
            }
        }

        return new ExtractionResult(nodes, edges);
    }

    // -----------------------------------------------------------------------
    // First-pass: collect stub class metadata
    // -----------------------------------------------------------------------

    private static class StubCollectorVisitor extends VoidVisitorAdapter<Void> {
        private final Set<String> soapStubFqns;
        private final Set<String> feignClientFqns;

        StubCollectorVisitor(Set<String> soapStubFqns, Set<String> feignClientFqns) {
            this.soapStubFqns = soapStubFqns;
            this.feignClientFqns = feignClientFqns;
        }

        @Override
        public void visit(ClassOrInterfaceDeclaration cls, Void arg) {
            String fqn = resolveClassFqn(cls);
            for (AnnotationExpr ann : cls.getAnnotations()) {
                if ("WebServiceClient".equals(ann.getNameAsString())) {
                    soapStubFqns.add(fqn);
                }
                if ("FeignClient".equals(ann.getNameAsString())) {
                    feignClientFqns.add(fqn);
                }
            }
            super.visit(cls, arg);
        }

        private static String resolveClassFqn(ClassOrInterfaceDeclaration cls) {
            try {
                return cls.resolve().getQualifiedName();
            } catch (Exception e) {
                Optional<CompilationUnit> cu = cls.findCompilationUnit();
                String pkg = cu.flatMap(c -> c.getPackageDeclaration())
                        .map(pd -> pd.getNameAsString()).orElse("");
                return pkg.isEmpty() ? cls.getNameAsString() : pkg + "." + cls.getNameAsString();
            }
        }
    }

    // -----------------------------------------------------------------------
    // Second-pass: find call sites
    // -----------------------------------------------------------------------

    private class CallSiteVisitor extends VoidVisitorAdapter<Void> {

        private final String repoId;
        private final String sha;
        private final Path sourceRoot;
        private final Set<String> soapStubFqns;
        private final Set<String> feignClientFqns;
        private final List<Map<String, Object>> nodes;
        private final List<Map<String, Object>> edges;

        // Context: currently visited class and method
        private String currentClassFqn = "";
        private String currentMethodFqn = "";

        CallSiteVisitor(String repoId, String sha, Path sourceRoot,
                        Set<String> soapStubFqns, Set<String> feignClientFqns,
                        List<Map<String, Object>> nodes, List<Map<String, Object>> edges) {
            this.repoId = repoId;
            this.sha = sha;
            this.sourceRoot = sourceRoot;
            this.soapStubFqns = soapStubFqns;
            this.feignClientFqns = feignClientFqns;
            this.nodes = nodes;
            this.edges = edges;
        }

        @Override
        public void visit(ClassOrInterfaceDeclaration cls, Void arg) {
            try {
                currentClassFqn = cls.resolve().getQualifiedName();
            } catch (Exception e) {
                Optional<CompilationUnit> cu = cls.findCompilationUnit();
                String pkg = cu.flatMap(c -> c.getPackageDeclaration())
                        .map(pd -> pd.getNameAsString()).orElse("");
                currentClassFqn = pkg.isEmpty() ? cls.getNameAsString() : pkg + "." + cls.getNameAsString();
            }
            super.visit(cls, arg);
        }

        @Override
        public void visit(MethodDeclaration method, Void arg) {
            currentMethodFqn = buildMethodFqn(method);
            super.visit(method, arg);
        }

        @Override
        public void visit(MethodCallExpr call, Void arg) {
            if (!currentMethodFqn.isEmpty()) {
                String methodName = call.getNameAsString();
                int line = call.getBegin().map(p -> p.line).orElse(1);
                String sourceRef = buildSourceRef(call);

                // ── RestTemplate call sites ──────────────────────────────────
                if (REST_TEMPLATE_METHODS.contains(methodName)) {
                    tryEmitRestTemplateEdge(call, methodName, line, sourceRef);
                }

                // ── WebClient call sites ─────────────────────────────────────
                if (WEB_CLIENT_METHODS.contains(methodName)) {
                    tryEmitWebClientEdge(call, methodName, line, sourceRef);
                }
            }
            super.visit(call, arg);
        }

        private void tryEmitRestTemplateEdge(MethodCallExpr call, String methodName,
                                              int line, String sourceRef) {
            try {
                // Check if receiver type resolves to RestTemplate
                Optional<String> receiverType = call.getScope()
                        .map(scope -> {
                            try {
                                return scope.calculateResolvedType().describe();
                            } catch (Exception ex) {
                                // Try simple name matching as fallback
                                return scope.toString().toLowerCase().contains("resttemplate")
                                        ? "org.springframework.web.client.RestTemplate" : null;
                            }
                        });

                boolean isRestTemplate = receiverType
                        .map(t -> t.contains("RestTemplate"))
                        .orElse(false);

                // Also check simple name fallback
                if (!isRestTemplate) {
                    isRestTemplate = call.getScope()
                            .map(s -> s.toString().toLowerCase().contains("resttemplate"))
                            .orElse(false);
                }

                if (isRestTemplate) {
                    String callerNodeId = NodeIdComputer.computeNodeId(repoId, currentMethodFqn, "METHOD");
                    String endpointNodeId = NodeIdComputer.urlEndpointNodeId(repoId, currentMethodFqn, line);
                    String endpointSourceRef = SourceRefBuilder.urlEndpoint(repoId, currentMethodFqn, line);
                    String httpMethod = toHttpMethod(methodName);

                    nodes.add(urlEndpointNode(endpointNodeId, endpointSourceRef, httpMethod, "rest_template"));
                    edges.add(callsRestEdge(callerNodeId, endpointNodeId, sourceRef,
                            httpMethod, "rest_template"));
                }
            } catch (Exception e) {
                LOG.debug("Could not resolve RestTemplate call at line {}: {}", line, e.getMessage());
            }
        }

        private void tryEmitWebClientEdge(MethodCallExpr call, String methodName,
                                           int line, String sourceRef) {
            try {
                boolean isWebClient = call.getScope()
                        .map(s -> {
                            try {
                                return s.calculateResolvedType().describe().contains("WebClient");
                            } catch (Exception ex) {
                                return s.toString().toLowerCase().contains("webclient");
                            }
                        })
                        .orElse(false);

                if (isWebClient) {
                    String callerNodeId = NodeIdComputer.computeNodeId(repoId, currentMethodFqn, "METHOD");
                    String endpointNodeId = NodeIdComputer.urlEndpointNodeId(repoId, currentMethodFqn, line);
                    String endpointSourceRef = SourceRefBuilder.urlEndpoint(repoId, currentMethodFqn, line);
                    String httpMethod = methodName.toUpperCase();

                    nodes.add(urlEndpointNode(endpointNodeId, endpointSourceRef, httpMethod, "web_client"));
                    edges.add(callsRestEdge(callerNodeId, endpointNodeId, sourceRef,
                            httpMethod, "web_client"));
                }
            } catch (Exception e) {
                LOG.debug("Could not resolve WebClient call at line {}: {}", line, e.getMessage());
            }
        }

        private String buildMethodFqn(MethodDeclaration method) {
            StringBuilder sb = new StringBuilder(currentClassFqn)
                    .append('#').append(method.getNameAsString()).append('(');
            List<com.github.javaparser.ast.body.Parameter> params = method.getParameters();
            for (int i = 0; i < params.size(); i++) {
                if (i > 0) sb.append(',');
                String typeName = params.get(i).getType().asString();
                try {
                    typeName = params.get(i).resolve().getType().describe();
                } catch (Exception ignored) {
                    // keep simple name
                }
                sb.append(typeName);
            }
            sb.append(')');
            return sb.toString();
        }

        private String buildSourceRef(com.github.javaparser.ast.Node node) {
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

        private static String toHttpMethod(String methodName) {
            return switch (methodName) {
                case "getForObject", "getForEntity" -> "GET";
                case "postForObject", "postForEntity" -> "POST";
                case "put" -> "PUT";
                case "delete" -> "DELETE";
                case "patchForObject" -> "PATCH";
                default -> "UNKNOWN";
            };
        }
    }

    // -----------------------------------------------------------------------
    // Node builders
    // -----------------------------------------------------------------------

    private Map<String, Object> feignClientNode(String fqn) {
        String nodeId = NodeIdComputer.computeNodeId(repoId, fqn, "INTERFACE");
        LinkedHashMap<String, Object> m = new LinkedHashMap<>();
        m.put("record_type", "node");
        m.put("node_id", nodeId);
        m.put("repo_id", repoId);
        m.put("qualified_name", fqn);
        m.put("kind", "INTERFACE");
        m.put("source_ref", SourceRefBuilder.build(repoId, sha, "unknown", 1));
        m.put("confidence", "inferred");
        m.put("phase_populated", 2);
        m.put("origin", "first_party");
        m.put("provenance_kind", "file");
        m.put("label", fqn.substring(fqn.lastIndexOf('.') + 1));
        return m;
    }

    private Map<String, Object> urlEndpointNode(String nodeId, String sourceRef,
                                                  String httpMethod, String clientType) {
        LinkedHashMap<String, Object> m = new LinkedHashMap<>();
        m.put("record_type", "node");
        m.put("node_id", nodeId);
        m.put("repo_id", repoId);
        m.put("kind", "URL_ENDPOINT");
        m.put("label", httpMethod + " unknown_endpoint");
        m.put("source_ref", sourceRef);
        m.put("confidence", "inferred");
        m.put("phase_populated", 2);
        m.put("origin", "external_stub");
        m.put("provenance_kind", "stub");
        m.put("http_client", clientType);
        m.put("url_pattern", "unknown");
        return m;
    }

    // -----------------------------------------------------------------------
    // Edge builders
    // -----------------------------------------------------------------------

    private Map<String, Object> callsSoapEdge(String fromNodeId, String toNodeId, String sourceRef) {
        LinkedHashMap<String, Object> m = new LinkedHashMap<>();
        m.put("record_type", "edge");
        m.put("edge_id", NodeIdComputer.computeEdgeId(fromNodeId, "CALLS_SOAP", toNodeId));
        m.put("label", "CALLS_SOAP");
        m.put("from_node_id", fromNodeId);
        m.put("to_node_id", toNodeId);
        m.put("source_ref", sourceRef);
        m.put("confidence", "inferred");
        m.put("tier", 3);
        m.put("phase_populated", 2);
        m.put("repo_id", repoId);
        m.put("evidence", "wsdl_stub_usage");
        m.put("completeness_caveat", CALLS_SOAP_CAVEAT);
        return m;
    }

    private Map<String, Object> callsRestEdge(String fromNodeId, String toNodeId,
                                               String sourceRef, String httpMethod,
                                               String httpClient) {
        LinkedHashMap<String, Object> m = new LinkedHashMap<>();
        m.put("record_type", "edge");
        m.put("edge_id", NodeIdComputer.computeEdgeId(fromNodeId, "CALLS_REST", toNodeId));
        m.put("label", "CALLS_REST");
        m.put("from_node_id", fromNodeId);
        m.put("to_node_id", toNodeId);
        m.put("source_ref", sourceRef);
        m.put("confidence", "inferred");
        m.put("tier", 3);
        m.put("phase_populated", 2);
        m.put("repo_id", repoId);
        m.put("evidence", "rest_client_usage");
        m.put("http_method", httpMethod);
        m.put("http_client", httpClient);
        m.put("path_pattern", "unknown");
        m.put("completeness_caveat", CALLS_REST_CAVEAT);
        return m;
    }
}
