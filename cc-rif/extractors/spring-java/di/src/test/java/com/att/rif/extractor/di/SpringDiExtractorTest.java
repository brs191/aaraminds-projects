package com.att.rif.extractor.di;

import com.att.rif.extractor.common.NodeIdComputer;
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
 * Unit tests for the Spring DI extractor.
 *
 * <h3>Fixture</h3>
 * Three classes written to a temp directory at test time:
 * <ul>
 *   <li>{@code CreditService} — {@code @Service} with an {@code @Autowired} field of type
 *       {@code CreditRepository}</li>
 *   <li>{@code CreditRepository} — {@code @Repository}</li>
 *   <li>{@code AppConfig} — {@code @Configuration} with a {@code @Bean} method returning
 *       {@code CreditService}</li>
 * </ul>
 *
 * <h3>Assertions</h3>
 * <ol>
 *   <li>INJECTS edge from CreditService → CreditRepository</li>
 *   <li>PRODUCES edge from AppConfig#creditService() → CreditService</li>
 *   <li>REGISTERS edges for all three classes → APPLICATION_CONTEXT</li>
 *   <li>All Tier-B edges have {@code confidence=probable}</li>
 *   <li>Determinism: two runs produce byte-identical sorted NDJSON</li>
 * </ol>
 */
class SpringDiExtractorTest {

    private static final String REPO_ID = "test-repo";
    private static final String SHA = "0000000000000000000000000000000000000001";
    private static final String PKG = "com.example.fixture";

    @TempDir
    Path tmpDir;

    private Path sourceRoot;

    @BeforeEach
    void writeFixtures() throws IOException {
        // Create package directory structure
        sourceRoot = tmpDir.resolve("src");
        Path pkgDir = sourceRoot.resolve("com/example/fixture");
        Files.createDirectories(pkgDir);

        // CreditService.java
        Files.writeString(pkgDir.resolve("CreditService.java"), """
                package com.example.fixture;

                import org.springframework.stereotype.Service;
                import org.springframework.beans.factory.annotation.Autowired;

                @Service
                public class CreditService {

                    @Autowired
                    private CreditRepository repo;
                }
                """);

        // CreditRepository.java
        Files.writeString(pkgDir.resolve("CreditRepository.java"), """
                package com.example.fixture;

                import org.springframework.stereotype.Repository;

                @Repository
                public class CreditRepository {
                }
                """);

        // AppConfig.java
        Files.writeString(pkgDir.resolve("AppConfig.java"), """
                package com.example.fixture;

                import org.springframework.context.annotation.Bean;
                import org.springframework.context.annotation.Configuration;

                @Configuration
                public class AppConfig {

                    @Bean
                    public CreditService creditService() {
                        return new CreditService();
                    }
                }
                """);
    }

    @Test
    void injectsEdgeFromCreditServiceToCreditRepository() throws Exception {
        List<Map<String, Object>> records = runExtractor();

        String creditServiceId = NodeIdComputer.computeNodeId(REPO_ID, PKG + ".CreditService", "CLASS");
        String creditRepoId    = NodeIdComputer.computeNodeId(REPO_ID, PKG + ".CreditRepository", "CLASS");

        List<Map<String, Object>> injectsEdges = findEdges(records, "INJECTS");
        assertFalse(injectsEdges.isEmpty(), "Expected at least one INJECTS edge");

        boolean found = injectsEdges.stream().anyMatch(e ->
                creditServiceId.equals(e.get("from_node_id")) &&
                creditRepoId.equals(e.get("to_node_id")));
        assertTrue(found,
                "Expected INJECTS edge from CreditService to CreditRepository.\n" +
                "INJECTS edges found: " + injectsEdges);
    }

    @Test
    void producesEdgeFromBeanMethodToCreditService() throws Exception {
        List<Map<String, Object>> records = runExtractor();

        // @Bean method FQN: com.example.fixture.AppConfig#creditService()
        String methodFqn    = PKG + ".AppConfig#creditService()";
        String methodNodeId = NodeIdComputer.computeNodeId(REPO_ID, methodFqn, "METHOD");
        String creditSvcId  = NodeIdComputer.computeNodeId(REPO_ID, PKG + ".CreditService", "CLASS");

        List<Map<String, Object>> producesEdges = findEdges(records, "PRODUCES");
        assertFalse(producesEdges.isEmpty(), "Expected at least one PRODUCES edge");

        boolean found = producesEdges.stream().anyMatch(e ->
                methodNodeId.equals(e.get("from_node_id")) &&
                creditSvcId.equals(e.get("to_node_id")));
        assertTrue(found,
                "Expected PRODUCES edge from AppConfig#creditService() to CreditService.\n" +
                "PRODUCES edges found: " + producesEdges);
    }

    @Test
    void registersEdgesForAllThreeClasses() throws Exception {
        List<Map<String, Object>> records = runExtractor();

        String appCtxId       = NodeIdComputer.applicationContextNodeId(REPO_ID);
        String creditSvcId    = NodeIdComputer.computeNodeId(REPO_ID, PKG + ".CreditService", "CLASS");
        String creditRepoId   = NodeIdComputer.computeNodeId(REPO_ID, PKG + ".CreditRepository", "CLASS");
        String appConfigId    = NodeIdComputer.computeNodeId(REPO_ID, PKG + ".AppConfig", "CLASS");

        List<Map<String, Object>> registersEdges = findEdges(records, "REGISTERS");

        assertRegistersEdge(registersEdges, creditSvcId,  appCtxId, "CreditService");
        assertRegistersEdge(registersEdges, creditRepoId, appCtxId, "CreditRepository");
        assertRegistersEdge(registersEdges, appConfigId,  appCtxId, "AppConfig");
    }

    @Test
    void applicationContextNodeIsEmitted() throws Exception {
        List<Map<String, Object>> records = runExtractor();
        String appCtxId = NodeIdComputer.applicationContextNodeId(REPO_ID);

        boolean found = records.stream()
                .filter(r -> "node".equals(r.get("record_type")))
                .anyMatch(r -> appCtxId.equals(r.get("node_id")));
        assertTrue(found, "Expected APPLICATION_CONTEXT node to be emitted");
    }

    @Test
    void allTierBEdgesHaveProbableConfidence() throws Exception {
        List<Map<String, Object>> records = runExtractor();
        List<Map<String, Object>> edges = records.stream()
                .filter(r -> "edge".equals(r.get("record_type")))
                .toList();
        assertFalse(edges.isEmpty(), "No edges emitted");
        for (Map<String, Object> edge : edges) {
            assertEquals("probable", edge.get("confidence"),
                    "Edge " + edge.get("label") + " should have confidence=probable, got: " + edge.get("confidence"));
        }
    }

    @Test
    void outputIsDeterministic() throws Exception {
        Path output1 = tmpDir.resolve("run1.ndjson");
        Path output2 = tmpDir.resolve("run2.ndjson");

        new SpringDiExtractor(REPO_ID, SHA, sourceRoot).extract();
        com.att.rif.extractor.common.EmitHelper.emit(
                new SpringDiExtractor(REPO_ID, SHA, sourceRoot).extract().nodes(),
                new SpringDiExtractor(REPO_ID, SHA, sourceRoot).extract().edges(),
                output1);
        com.att.rif.extractor.common.EmitHelper.emit(
                new SpringDiExtractor(REPO_ID, SHA, sourceRoot).extract().nodes(),
                new SpringDiExtractor(REPO_ID, SHA, sourceRoot).extract().edges(),
                output2);

        String content1 = Files.readString(output1);
        String content2 = Files.readString(output2);
        assertEquals(content1, content2, "Two runs on the same fixture must produce identical output");
    }

    // ── Helpers ─────────────────────────────────────────────────────────────

    private List<Map<String, Object>> runExtractor() throws Exception {
        SpringDiExtractor extractor = new SpringDiExtractor(REPO_ID, SHA, sourceRoot);
        SpringDiExtractor.ExtractionResult result = extractor.extract();
        Path outFile = tmpDir.resolve("output.ndjson");
        com.att.rif.extractor.common.EmitHelper.emit(result.nodes(), result.edges(), outFile);
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

    private void assertRegistersEdge(List<Map<String, Object>> registersEdges,
                                      String fromNodeId, String toNodeId, String className) {
        boolean found = registersEdges.stream().anyMatch(e ->
                fromNodeId.equals(e.get("from_node_id")) &&
                toNodeId.equals(e.get("to_node_id")));
        assertTrue(found,
                "Expected REGISTERS edge from " + className + " to APPLICATION_CONTEXT.\n" +
                "REGISTERS edges found: " + registersEdges);
    }
}
