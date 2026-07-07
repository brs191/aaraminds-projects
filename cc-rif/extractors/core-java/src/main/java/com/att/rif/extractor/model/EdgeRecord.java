package com.att.rif.extractor.model;

import com.att.rif.extractor.resolve.NodeIdComputer;
import java.util.LinkedHashMap;
import java.util.Map;

public final class EdgeRecord {
    public static final String IMPORTS_CAVEAT = "Wildcard imports (import com.example.*) are invisible to Tier-A extraction. Types brought in by wildcard are not linked until Phase 2 SCIP resolution. Static imports of members produce no edge. Types used solely via reflection or Class.forName() are not captured.";
    public static final String SAME_FILE_CALLS_CAVEAT = "Only intra-file static calls resolvable by JavaParser SymbolSolver are captured. Cross-file calls are not modelled in Tier A. Calls via interface reference, reflection, or dynamic dispatch whose concrete target is in a different file are absent. Lambda and method-reference call-sites are resolved only when SymbolSolver can fully resolve the target type; unresolved lambdas produce no edge. Calls to Lombok-generated methods produce no edge because no callee node exists in Phase 1. SymbolSolver overload resolution failures are silently dropped.";
    public static final String EXTENDS_CAVEAT = "Only the immediate supertype is captured. If the supertype is in an external dependency JAR, the to_node_id targets a stub node with confidence=probable and phase_populated=2.";
    public static final String IMPLEMENTS_CAVEAT = "Only direct interface implementations are captured. If the interface is not resolvable in the local source tree, a stub node is created with confidence=probable and phase_populated=2.";
    public static final String DECLARES_FIELD_CAVEAT = "Lombok-generated fields (e.g. @Data, @Value, @Builder) do not appear as source AST nodes and produce no DECLARES_FIELD edge in Phase 1. Enum constants are not modelled as FIELD nodes.";

    private EdgeRecord() {
    }

    public static Map<String, Object> imports(String fromNodeId, String toNodeId, String sourceRef) {
        return common("IMPORTS", fromNodeId, toNodeId, sourceRef, IMPORTS_CAVEAT);
    }

    public static Map<String, Object> sameFileCalls(String fromNodeId, String toNodeId, String sourceRef) {
        return common("SAME_FILE_CALLS", fromNodeId, toNodeId, sourceRef, SAME_FILE_CALLS_CAVEAT);
    }

    public static Map<String, Object> extendsEdge(String fromNodeId, String toNodeId, String sourceRef) {
        return common("EXTENDS", fromNodeId, toNodeId, sourceRef, EXTENDS_CAVEAT);
    }

    public static Map<String, Object> implementsEdge(String fromNodeId, String toNodeId, String sourceRef) {
        return common("IMPLEMENTS", fromNodeId, toNodeId, sourceRef, IMPLEMENTS_CAVEAT);
    }

    public static Map<String, Object> declaresField(String fromNodeId, String toNodeId, String sourceRef) {
        return common("DECLARES_FIELD", fromNodeId, toNodeId, sourceRef, DECLARES_FIELD_CAVEAT);
    }

    private static Map<String, Object> common(String label, String fromNodeId, String toNodeId, String sourceRef, String caveat) {
        LinkedHashMap<String, Object> map = new LinkedHashMap<>();
        map.put("record_type", "edge");
        map.put("edge_id", NodeIdComputer.computeEdgeId(fromNodeId, label, toNodeId));
        map.put("label", label);
        map.put("from_node_id", fromNodeId);
        map.put("to_node_id", toNodeId);
        map.put("confidence", "exact");
        map.put("source_ref", sourceRef);
        map.put("tier", 1);
        map.put("phase_populated", 1);
        map.put("completeness_caveat", caveat);
        return map;
    }
}
