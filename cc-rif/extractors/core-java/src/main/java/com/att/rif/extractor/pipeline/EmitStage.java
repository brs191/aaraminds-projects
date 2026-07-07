package com.att.rif.extractor.pipeline;

import com.att.rif.extractor.ExtractorConfig;
import com.att.rif.extractor.model.RunMetrics;
import com.fasterxml.jackson.databind.ObjectMapper;
import java.io.IOException;
import java.io.Writer;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.Comparator;
import java.util.List;
import java.util.Map;

public class EmitStage {
    private final ObjectMapper objectMapper = new ObjectMapper();
    @SuppressWarnings("unused")
    private final ExtractorConfig config;
    private final RunMetrics metrics;

    public EmitStage(ExtractorConfig config, RunMetrics metrics) {
        this.config = config;
        this.metrics = metrics;
    }

    public void emit(ExtractionResult result, Path outputFile) throws IOException {
        List<Map<String, Object>> nodes = result.nodes().stream()
                .sorted(Comparator.comparing(record -> String.valueOf(record.get("node_id"))))
                .toList();
        List<Map<String, Object>> edges = result.edges().stream()
                .sorted(Comparator.comparing(record -> String.valueOf(record.get("edge_id"))))
                .toList();

        if (outputFile.getParent() != null) {
            Files.createDirectories(outputFile.getParent());
        }
        try (Writer writer = Files.newBufferedWriter(outputFile, StandardCharsets.UTF_8)) {
            for (Map<String, Object> node : nodes) {
                writer.write(objectMapper.writeValueAsString(node));
                writer.write('\n');
            }
            for (Map<String, Object> edge : edges) {
                writer.write(objectMapper.writeValueAsString(edge));
                writer.write('\n');
            }
        }
        metrics.setNodesEmitted(nodes.size());
        metrics.setEdgesEmitted(edges.size());
    }
}
