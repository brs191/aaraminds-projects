package com.aaraminds.rif.extractor.pipeline;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;

public class ExtractionResult {
    private final List<Map<String, Object>> nodes = new ArrayList<>();
    private final List<Map<String, Object>> edges = new ArrayList<>();

    public void addNode(Map<String, Object> node) {
        nodes.add(node);
    }

    public void addNodes(List<Map<String, Object>> newNodes) {
        nodes.addAll(newNodes);
    }

    public void addEdge(Map<String, Object> edge) {
        edges.add(edge);
    }

    public void addEdges(List<Map<String, Object>> newEdges) {
        edges.addAll(newEdges);
    }

    public List<Map<String, Object>> nodes() {
        return nodes;
    }

    public List<Map<String, Object>> edges() {
        return edges;
    }
}
