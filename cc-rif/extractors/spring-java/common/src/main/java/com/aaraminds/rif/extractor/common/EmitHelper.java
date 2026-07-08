package com.aaraminds.rif.extractor.common;

import com.fasterxml.jackson.databind.ObjectMapper;
import java.io.IOException;
import java.io.Writer;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.Comparator;
import java.util.List;
import java.util.Map;

/**
 * Writes Phase 2 extraction results to an NDJSON file.
 * <p>
 * Ordering contract (matches Phase 1 EmitStage):
 * <ol>
 *   <li>All node records first, sorted lexicographically by {@code node_id}.</li>
 *   <li>All edge records after, sorted lexicographically by {@code edge_id}.</li>
 * </ol>
 * This ordering guarantees deterministic, byte-identical output for the same input commit.
 */
public final class EmitHelper {

    private static final ObjectMapper MAPPER = new ObjectMapper();

    private EmitHelper() {
    }

    /**
     * Writes nodes (sorted by node_id) then edges (sorted by edge_id) to {@code outputFile}.
     * Creates parent directories as needed.
     */
    public static void emit(List<Map<String, Object>> nodes,
                             List<Map<String, Object>> edges,
                             Path outputFile) throws IOException {
        List<Map<String, Object>> sortedNodes = nodes.stream()
                .sorted(Comparator.comparing(r -> String.valueOf(r.get("node_id"))))
                .toList();
        List<Map<String, Object>> sortedEdges = edges.stream()
                .sorted(Comparator.comparing(r -> String.valueOf(r.get("edge_id"))))
                .toList();

        if (outputFile.getParent() != null) {
            Files.createDirectories(outputFile.getParent());
        }
        try (Writer writer = Files.newBufferedWriter(outputFile, StandardCharsets.UTF_8)) {
            for (Map<String, Object> node : sortedNodes) {
                writer.write(MAPPER.writeValueAsString(node));
                writer.write('\n');
            }
            for (Map<String, Object> edge : sortedEdges) {
                writer.write(MAPPER.writeValueAsString(edge));
                writer.write('\n');
            }
        }
    }
}
