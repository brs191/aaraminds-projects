package com.aaraminds.rif.extractor;

import com.aaraminds.rif.extractor.model.RunMetrics;
import com.aaraminds.rif.extractor.pipeline.DiscoveryStage;
import com.aaraminds.rif.extractor.pipeline.EmitStage;
import com.aaraminds.rif.extractor.pipeline.ExtractionResult;
import com.aaraminds.rif.extractor.pipeline.ParseStage;
import com.aaraminds.rif.extractor.pipeline.ResolveStage;
import com.aaraminds.rif.extractor.resolve.NodeIdComputer;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.github.javaparser.ParserConfiguration;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

class DeterminismTest {
    private final List<Path> tempFiles = new ArrayList<>();
    private final ObjectMapper objectMapper = new ObjectMapper();

    @AfterEach
    void cleanup() throws IOException {
        for (Path tempFile : tempFiles) {
            Files.deleteIfExists(tempFile);
        }
    }

    @Test
    void emitsByteIdenticalOutputAndValidNdjson() throws Exception {
        Path fixtureDir = Path.of("src/test/resources/fixtures/simple");
        Path first = Files.createTempFile("rif-test-first", ".ndjson");
        Path second = Files.createTempFile("rif-test-second", ".ndjson");
        tempFiles.add(first);
        tempFiles.add(second);

        runPipeline(fixtureDir, first);
        runPipeline(fixtureDir, second);

        String firstHash = NodeIdComputer.sha256(Files.readString(first, StandardCharsets.UTF_8));
        String secondHash = NodeIdComputer.sha256(Files.readString(second, StandardCharsets.UTF_8));
        assertEquals(firstHash, secondHash);

        List<String> lines = Files.readAllLines(first, StandardCharsets.UTF_8);
        boolean seenEdge = false;
        for (String line : lines) {
            @SuppressWarnings("unchecked")
            Map<String, Object> record = objectMapper.readValue(line, Map.class);
            String recordType = String.valueOf(record.get("record_type"));
            if ("edge".equals(recordType)) {
                seenEdge = true;
            }
            if ("node".equals(recordType)) {
                assertTrue(!seenEdge, "Node appeared after edge");
            }
        }
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
