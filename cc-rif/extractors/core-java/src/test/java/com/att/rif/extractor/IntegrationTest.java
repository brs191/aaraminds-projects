package com.att.rif.extractor;

import com.att.rif.extractor.model.RunMetrics;
import com.att.rif.extractor.pipeline.DiscoveryStage;
import com.att.rif.extractor.pipeline.EmitStage;
import com.att.rif.extractor.pipeline.ExtractionResult;
import com.att.rif.extractor.pipeline.ParseStage;
import com.att.rif.extractor.pipeline.ResolveStage;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.github.javaparser.ParserConfiguration;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.regex.Pattern;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertTrue;

class IntegrationTest {
    private static final Pattern FIRST_PARTY = Pattern.compile("^[a-zA-Z0-9._/-]+@[0-9a-f]{40}:[^:]+:[1-9][0-9]*$");

    private final List<Path> tempFiles = new ArrayList<>();
    private final ObjectMapper objectMapper = new ObjectMapper();

    @AfterEach
    void cleanup() throws IOException {
        for (Path tempFile : tempFiles) {
            Files.deleteIfExists(tempFile);
        }
    }

    @Test
    void extractsExpectedGraphForSpringFixture() throws Exception {
        Path output = Files.createTempFile("rif-test-spring", ".ndjson");
        tempFiles.add(output);
        runPipeline(Path.of("src/test/resources/fixtures/spring"), output);

        int fileNodes = 0;
        int classNodes = 0;
        int interfaceNodes = 0;
        int methodNodes = 0;
        int constructorNodes = 0;
        int fieldNodes = 0;
        int importEdges = 0;
        int declaresFieldEdges = 0;
        int sameFileCalls = 0;
        int totalRecords = 0;
        boolean creditServiceLombok = false;
        boolean creditRequestLombok = false;

        for (String line : Files.readAllLines(output, StandardCharsets.UTF_8)) {
            totalRecords++;
            @SuppressWarnings("unchecked")
            Map<String, Object> record = objectMapper.readValue(line, Map.class);
            if ("node".equals(record.get("record_type"))) {
                String kind = String.valueOf(record.get("kind"));
                switch (kind) {
                    case "FILE" -> fileNodes++;
                    case "CLASS", "RECORD" -> classNodes++;
                    case "INTERFACE" -> interfaceNodes++;
                    case "METHOD" -> methodNodes++;
                    case "CONSTRUCTOR" -> constructorNodes++;
                    case "FIELD" -> fieldNodes++;
                    default -> {
                    }
                }
                if ("com.example.spring.CreditService".equals(record.get("qualified_name")) && Boolean.TRUE.equals(record.get("lombok_present"))) {
                    creditServiceLombok = true;
                }
                if ("com.example.spring.dto.CreditRequest".equals(record.get("qualified_name")) && Boolean.TRUE.equals(record.get("lombok_present"))) {
                    creditRequestLombok = true;
                }
                if ("first_party".equals(record.get("origin")) && "file".equals(record.get("provenance_kind"))) {
                    assertTrue(FIRST_PARTY.matcher(String.valueOf(record.get("source_ref"))).matches());
                }
            }
            if ("edge".equals(record.get("record_type"))) {
                String label = String.valueOf(record.get("label"));
                switch (label) {
                    case "IMPORTS" -> importEdges++;
                    case "DECLARES_FIELD" -> declaresFieldEdges++;
                    case "SAME_FILE_CALLS" -> sameFileCalls++;
                    default -> {
                    }
                }
            }
        }

        assertTrue(fileNodes == 5, "Expected 5 FILE nodes");
        assertTrue(classNodes >= 3, "Expected at least 3 CLASS/RECORD nodes");
        assertTrue(interfaceNodes >= 1, "Expected at least 1 INTERFACE node");
        assertTrue(methodNodes >= 4, "Expected at least 4 METHOD nodes");
        assertTrue(constructorNodes >= 1, "Expected at least 1 CONSTRUCTOR node");
        assertTrue(fieldNodes >= 2, "Expected at least 2 FIELD nodes");
        assertTrue(importEdges >= 3, "Expected at least 3 IMPORTS edges");
        assertTrue(declaresFieldEdges >= 2, "Expected at least 2 DECLARES_FIELD edges");
        assertTrue(sameFileCalls >= 1, "Expected at least 1 SAME_FILE_CALLS edge");
        assertTrue(totalRecords >= 20, "Expected at least 20 total records");
        assertTrue(creditServiceLombok, "CreditService should be marked as Lombok present");
        assertTrue(creditRequestLombok, "CreditRequest should be marked as Lombok present");
    }

    private void runPipeline(Path fixtureDir, Path outputFile) throws Exception {
        ExtractorConfig config = new ExtractorConfig(fixtureDir, "test-repo", "a".repeat(40), outputFile, false, null);
        RunMetrics metrics = new RunMetrics();
        DiscoveryStage discoveryStage = new DiscoveryStage(config, metrics);
        List<Path> files = discoveryStage.discover();
        ParseStage parseStage = new ParseStage(config, metrics);
        ParserConfiguration parserConfiguration = parseStage.buildParserConfiguration();
        ResolveStage resolveStage = new ResolveStage(config, metrics, parserConfiguration);
        ExtractionResult result = resolveStage.resolve(files);
        EmitStage emitStage = new EmitStage(config, metrics);
        emitStage.emit(result, outputFile);
    }
}
