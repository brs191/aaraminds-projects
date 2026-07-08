package com.aaraminds.rif.extractor;

import com.aaraminds.rif.extractor.model.RunMetrics;
import com.aaraminds.rif.extractor.pipeline.DiscoveryStage;
import com.aaraminds.rif.extractor.pipeline.EmitStage;
import com.aaraminds.rif.extractor.pipeline.ExtractionResult;
import com.aaraminds.rif.extractor.pipeline.ParseStage;
import com.aaraminds.rif.extractor.pipeline.ResolveStage;
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

import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.junit.jupiter.api.Assertions.assertTrue;

class ProvenanceTest {
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
    void emitsNonEmptySourceRefsWithCorrectFormats() throws Exception {
        Path output = Files.createTempFile("rif-test-provenance", ".ndjson");
        tempFiles.add(output);
        runPipeline(Path.of("src/test/resources/fixtures/simple"), output);

        for (String line : Files.readAllLines(output, StandardCharsets.UTF_8)) {
            @SuppressWarnings("unchecked")
            Map<String, Object> record = objectMapper.readValue(line, Map.class);
            Object sourceRef = record.get("source_ref");
            assertNotNull(sourceRef);
            assertFalse(String.valueOf(sourceRef).isEmpty());
            if ("first_party".equals(record.get("origin")) && "file".equals(record.get("provenance_kind"))) {
                assertTrue(FIRST_PARTY.matcher(String.valueOf(sourceRef)).matches(), () -> "Unexpected source_ref: " + sourceRef);
            }
            if ("stub".equals(record.get("provenance_kind"))) {
                assertTrue(String.valueOf(sourceRef).startsWith("STUB:external:"));
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
