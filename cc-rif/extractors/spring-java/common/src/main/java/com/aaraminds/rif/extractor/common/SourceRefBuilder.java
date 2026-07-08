package com.aaraminds.rif.extractor.common;

/**
 * Builds {@code source_ref} strings in the canonical format:
 * <pre>{repoId}@{sha}:{relativePath}:{line}</pre>
 *
 * Synthetic nodes use:
 * <pre>SYNTHETIC:{repoId}:{tag}[:{extra}]</pre>
 *
 * Matches Phase 1 SourceRefBuilder contract exactly.
 */
public final class SourceRefBuilder {

    private SourceRefBuilder() {
    }

    /** Standard source_ref for a real source location. */
    public static String build(String repoId, String sha, String relativePath, int line) {
        return repoId + "@" + sha + ":" + relativePath + ":" + line;
    }

    /** source_ref for APPLICATION_CONTEXT virtual node (§1.2). */
    public static String applicationContext(String repoId) {
        return "STUB:virtual:APPLICATION_CONTEXT:" + repoId;
    }

    /** source_ref for a POINTCUT_EXPRESSION synthetic node. */
    public static String pointcutExpression(String repoId, String aspectClassFqn, int line) {
        return "SYNTHETIC:" + repoId + ":POINTCUT:" + aspectClassFqn + ":" + line;
    }

    /** source_ref for a URL_ENDPOINT synthetic node. */
    public static String urlEndpoint(String repoId, String callingMethodFqn, int line) {
        return "SYNTHETIC:" + repoId + ":URL_ENDPOINT:" + callingMethodFqn + ":" + line;
    }
}
