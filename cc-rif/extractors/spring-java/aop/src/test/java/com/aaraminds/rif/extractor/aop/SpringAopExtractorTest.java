package com.aaraminds.rif.extractor.aop;

import com.aaraminds.rif.extractor.common.EmitHelper;
import com.aaraminds.rif.extractor.common.NodeIdComputer;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Unit tests for the Spring AOP extractor.
 *
 * <h3>Fixture</h3>
 * {@code LoggingAspect} — an {@code @Aspect @Component} class with an {@code @Around}
 * advice method carrying a pointcut expression.
 *
 * <h3>Assertions</h3>
 * <ol>
 *   <li>An {@code ADVISES} edge is emitted.</li>
 *   <li>The edge has {@code pointcut_expr = "execution(* com.example.service.*.*(..))"} (exact).</li>
 *   <li>The edge has {@code completeness_caveat = "static pointcut match only — runtime proxy targets may differ"}.</li>
 *   <li>The edge has {@code confidence = "inferred"} (Tier-C).</li>
 *   <li>A synthetic {@code POINTCUT_EXPRESSION} node is emitted.</li>
 *   <li>Determinism: two runs on the same fixture produce byte-identical output.</li>
 * </ol>
 */
class SpringAopExtractorTest {

    private static final String REPO_ID   = "test-aop-repo";
    private static final String SHA       = "0000000000000000000000000000000000000002";
    private static final String PKG       = "com.example.aspect";
    private static final String POINTCUT  = "execution(* com.example.service.*.*(..))";

    @TempDir
    Path tmpDir;

    private Path sourceRoot;

    @BeforeEach
    void writeFixtures() throws IOException {
        sourceRoot = tmpDir.resolve("src");
        Path pkgDir = sourceRoot.resolve("com/example/aspect");
        Files.createDirectories(pkgDir);

        // LoggingAspect.java
        Files.writeString(pkgDir.resolve("LoggingAspect.java"), """
                package com.example.aspect;

                import org.aspectj.lang.ProceedingJoinPoint;
                import org.aspectj.lang.annotation.Around;
                import org.aspectj.lang.annotation.Aspect;
                import org.springframework.stereotype.Component;

                @Aspect
                @Component
                public class LoggingAspect {

                    @Around("execution(* com.example.service.*.*(..))")
                    public Object logAround(ProceedingJoinPoint jp) throws Throwable {
                        return jp.proceed();
                    }
                }
                """);
    }

    @Test
    void advisesEdgeIsEmitted() throws Exception {
        List<Map<String, Object>> records = runExtractor();
        List<Map<String, Object>> advisesEdges = findEdges(records, "ADVISES");
        assertFalse(advisesEdges.isEmpty(), "Expected at least one ADVISES edge");
    }

    @Test
    void advisesEdgeHasCorrectPointcutExpr() throws Exception {
        List<Map<String, Object>> records = runExtractor();
        List<Map<String, Object>> advisesEdges = findEdges(records, "ADVISES");
        assertFalse(advisesEdges.isEmpty(), "Expected at least one ADVISES edge");

        Map<String, Object> edge = advisesEdges.get(0);
        assertEquals(POINTCUT, edge.get("pointcut_expr"),
                "pointcut_expr must match the annotation value exactly");
    }

    @Test
    void advisesEdgeHasCorrectCompletenessCaveat() throws Exception {
        List<Map<String, Object>> records = runExtractor();
        List<Map<String, Object>> advisesEdges = findEdges(records, "ADVISES");
        assertFalse(advisesEdges.isEmpty(), "Expected at least one ADVISES edge");

        Map<String, Object> edge = advisesEdges.get(0);
        assertEquals(SpringAopExtractor.ADVISES_CAVEAT, edge.get("completeness_caveat"),
                "completeness_caveat must match the spec exactly");
    }

    @Test
    void advisesEdgeHasInferredConfidence() throws Exception {
        List<Map<String, Object>> records = runExtractor();
        List<Map<String, Object>> advisesEdges = findEdges(records, "ADVISES");
        assertFalse(advisesEdges.isEmpty(), "Expected at least one ADVISES edge");

        Map<String, Object> edge = advisesEdges.get(0);
        assertEquals("inferred", edge.get("confidence"),
                "ADVISES edge must have confidence=inferred (Tier-C)");
    }

    @Test
    void advisesEdgeFromAspectClassToPointcutNode() throws Exception {
        List<Map<String, Object>> records = runExtractor();

        String aspectClassId = NodeIdComputer.computeNodeId(REPO_ID, PKG + ".LoggingAspect", "CLASS");
        List<Map<String, Object>> advisesEdges = findEdges(records, "ADVISES");
        assertFalse(advisesEdges.isEmpty());

        Map<String, Object> edge = advisesEdges.get(0);
        assertEquals(aspectClassId, edge.get("from_node_id"),
                "ADVISES edge must originate from the @Aspect CLASS node");

        // The to_node_id must reference a POINTCUT_EXPRESSION node present in the output
        String toNodeId = (String) edge.get("to_node_id");
        boolean pointcutNodePresent = records.stream()
                .filter(r -> "node".equals(r.get("record_type")))
                .anyMatch(r -> toNodeId.equals(r.get("node_id")) &&
                               "POINTCUT_EXPRESSION".equals(r.get("kind")));
        assertTrue(pointcutNodePresent,
                "A POINTCUT_EXPRESSION node with node_id=" + toNodeId + " must be emitted");
    }

    @Test
    void advisesEdgeHasAroundAdviceType() throws Exception {
        List<Map<String, Object>> records = runExtractor();
        List<Map<String, Object>> advisesEdges = findEdges(records, "ADVISES");
        assertFalse(advisesEdges.isEmpty());

        Map<String, Object> edge = advisesEdges.get(0);
        assertEquals("around", edge.get("advice_type"),
                "Advice type must be 'around' for @Around annotation");
    }

    @Test
    void outputIsDeterministic() throws Exception {
        Path output1 = tmpDir.resolve("run1.ndjson");
        Path output2 = tmpDir.resolve("run2.ndjson");

        SpringAopExtractor.ExtractionResult r1 = new SpringAopExtractor(REPO_ID, SHA, sourceRoot).extract();
        EmitHelper.emit(r1.nodes(), r1.edges(), output1);

        SpringAopExtractor.ExtractionResult r2 = new SpringAopExtractor(REPO_ID, SHA, sourceRoot).extract();
        EmitHelper.emit(r2.nodes(), r2.edges(), output2);

        String content1 = Files.readString(output1);
        String content2 = Files.readString(output2);
        assertEquals(content1, content2, "Two runs on the same fixture must produce identical output");
    }

    // ── Helpers ─────────────────────────────────────────────────────────────

    private List<Map<String, Object>> runExtractor() throws Exception {
        SpringAopExtractor extractor = new SpringAopExtractor(REPO_ID, SHA, sourceRoot);
        SpringAopExtractor.ExtractionResult result = extractor.extract();
        Path outFile = tmpDir.resolve("output.ndjson");
        EmitHelper.emit(result.nodes(), result.edges(), outFile);
        return parseNdjson(outFile);
    }

    private List<Map<String, Object>> parseNdjson(Path file) throws IOException {
        ObjectMapper mapper = new ObjectMapper();
        TypeReference<Map<String, Object>> mapType = new TypeReference<>() {};
        return Files.readAllLines(file).stream()
                .filter(line -> !line.isBlank())
                .map(line -> {
                    try {
                        return mapper.readValue(line, mapType);
                    } catch (Exception e) {
                        throw new RuntimeException("Failed to parse NDJSON line: " + line, e);
                    }
                })
                .collect(Collectors.toList());
    }

    private List<Map<String, Object>> findEdges(List<Map<String, Object>> records, String label) {
        return records.stream()
                .filter(r -> "edge".equals(r.get("record_type")) && label.equals(r.get("label")))
                .collect(Collectors.toList());
    }
}
