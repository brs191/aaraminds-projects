package com.att.rif.extractor.model;

import com.att.rif.extractor.resolve.NodeIdComputer;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

public final class NodeRecord {
    private NodeRecord() {
    }

    public static Map<String, Object> fileNode(
            String repoId,
            String qualifiedName,
            String sourceRef,
            String packageName,
            int lineCount) {
        LinkedHashMap<String, Object> map = common(repoId, qualifiedName, "FILE", sourceRef, "exact", "first_party", "file");
        map.put("package", packageName);
        map.put("line_count", lineCount);
        return map;
    }

    public static Map<String, Object> classNode(
            String repoId,
            String qualifiedName,
            String kind,
            String sourceRef,
            String simpleName,
            boolean isAbstract,
            boolean isInner,
            List<String> annotations,
            boolean lombokPresent,
            List<String> permitsTypes,
            String origin,
            String provenanceKind,
            String confidence) {
        LinkedHashMap<String, Object> map = common(repoId, qualifiedName, kind, sourceRef, confidence, origin, provenanceKind);
        map.put("simple_name", simpleName);
        map.put("is_abstract", isAbstract);
        map.put("is_inner", isInner);
        map.put("annotations", annotations);
        map.put("lombok_present", lombokPresent);
        map.put("permits_types", permitsTypes);
        return map;
    }

    public static Map<String, Object> interfaceNode(
            String repoId,
            String qualifiedName,
            String sourceRef,
            String simpleName,
            boolean isInner,
            List<String> annotations,
            boolean lombokPresent,
            List<String> permitsTypes,
            String origin,
            String provenanceKind,
            String confidence) {
        LinkedHashMap<String, Object> map = common(repoId, qualifiedName, "INTERFACE", sourceRef, confidence, origin, provenanceKind);
        map.put("simple_name", simpleName);
        map.put("is_inner", isInner);
        map.put("annotations", annotations);
        map.put("lombok_present", lombokPresent);
        map.put("permits_types", permitsTypes);
        return map;
    }

    public static Map<String, Object> enumNode(
            String repoId,
            String qualifiedName,
            String sourceRef,
            String simpleName,
            boolean isInner,
            List<String> annotations,
            List<String> constants,
            boolean lombokPresent,
            String origin,
            String provenanceKind,
            String confidence) {
        LinkedHashMap<String, Object> map = common(repoId, qualifiedName, "ENUM", sourceRef, confidence, origin, provenanceKind);
        map.put("simple_name", simpleName);
        map.put("is_inner", isInner);
        map.put("annotations", annotations);
        map.put("constants", constants);
        map.put("lombok_present", lombokPresent);
        return map;
    }

    public static Map<String, Object> methodNode(
            String repoId,
            String qualifiedName,
            String kind,
            String sourceRef,
            String simpleName,
            String returnType,
            List<String> paramTypes,
            boolean isStatic,
            String visibility,
            List<String> annotations) {
        LinkedHashMap<String, Object> map = common(repoId, qualifiedName, kind, sourceRef, "exact", "first_party", "file");
        map.put("simple_name", simpleName);
        if ("METHOD".equals(kind)) {
            map.put("return_type", returnType);
        }
        map.put("param_types", paramTypes);
        map.put("is_static", isStatic);
        map.put("visibility", visibility);
        map.put("annotations", annotations);
        map.put("scip_symbol", null);
        return map;
    }

    public static Map<String, Object> fieldNode(
            String repoId,
            String qualifiedName,
            String sourceRef,
            String simpleName,
            String typeName,
            boolean isStatic,
            boolean isFinal,
            String visibility,
            List<String> annotations) {
        LinkedHashMap<String, Object> map = common(repoId, qualifiedName, "FIELD", sourceRef, "exact", "first_party", "file");
        map.put("simple_name", simpleName);
        map.put("type_name", typeName);
        map.put("is_static", isStatic);
        map.put("is_final", isFinal);
        map.put("visibility", visibility);
        map.put("annotations", annotations);
        map.put("lombok_generated", false);
        return map;
    }

    private static LinkedHashMap<String, Object> common(
            String repoId,
            String qualifiedName,
            String kind,
            String sourceRef,
            String confidence,
            String origin,
            String provenanceKind) {
        LinkedHashMap<String, Object> map = new LinkedHashMap<>();
        map.put("record_type", "node");
        map.put("node_id", NodeIdComputer.computeNodeId(repoId, qualifiedName, kind));
        map.put("repo_id", repoId);
        map.put("qualified_name", qualifiedName);
        map.put("kind", kind);
        map.put("source_ref", sourceRef);
        map.put("confidence", confidence);
        map.put("phase_populated", 1);
        map.put("origin", origin);
        map.put("provenance_kind", provenanceKind);
        return map;
    }
}
