package com.att.rif.extractor;

import java.nio.file.Path;
import java.util.List;

public record ExtractorConfig(
        Path repoPath,
        String repoId,
        String sha,
        Path output,
        boolean skipTests,
        Path depsPath,
        boolean verbose,
        List<Path> filesFilter) {

    public ExtractorConfig {
        repoPath = repoPath.toAbsolutePath().normalize();
        output = output.toAbsolutePath().normalize();
        depsPath = depsPath == null ? null : depsPath.toAbsolutePath().normalize();
        filesFilter = filesFilter == null
                ? List.of()
                : filesFilter.stream().map(path -> path.normalize()).toList();
    }

    public ExtractorConfig(Path repoPath, String repoId, String sha, Path output, boolean skipTests, Path depsPath) {
        this(repoPath, repoId, sha, output, skipTests, depsPath, false, List.of());
    }
}
