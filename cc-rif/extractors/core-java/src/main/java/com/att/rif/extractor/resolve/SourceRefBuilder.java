package com.att.rif.extractor.resolve;

public final class SourceRefBuilder {
    public static final String UNAVAILABLE = "UNAVAILABLE:no-position";

    private SourceRefBuilder() {
    }

    public static String build(String repoId, String sha, String relativePath, int line) {
        return repoId + "@" + sha + ":" + relativePath + ":" + line;
    }

    public static String unavailable() {
        return UNAVAILABLE;
    }
}
