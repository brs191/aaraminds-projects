package com.att.rif.extractor.model;

import com.att.rif.extractor.resolve.NodeIdComputer;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

public class StubNode {
    private final ConcurrentHashMap<String, Map<String, Object>> stubs = new ConcurrentHashMap<>();

    public Map<String, Object> getOrCreate(String repoId, String fqn, String kind) {
        String nodeId = NodeIdComputer.computeNodeId(repoId, fqn, kind);
        return stubs.computeIfAbsent(nodeId, ignored -> NodeRecord.classNode(
                repoId,
                fqn,
                kind,
                "STUB:external:" + fqn,
                simpleName(fqn),
                false,
                false,
                List.of(),
                false,
                null,
                "external_stub",
                "stub",
                "probable"));
    }

    public List<Map<String, Object>> all() {
        return new ArrayList<>(stubs.values());
    }

    private static String simpleName(String fqn) {
        int lastDot = fqn.lastIndexOf('.');
        return lastDot >= 0 ? fqn.substring(lastDot + 1) : fqn;
    }
}
