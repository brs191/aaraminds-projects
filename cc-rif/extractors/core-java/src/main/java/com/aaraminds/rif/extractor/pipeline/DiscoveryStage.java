package com.aaraminds.rif.extractor.pipeline;

import com.aaraminds.rif.extractor.ExtractorConfig;
import com.aaraminds.rif.extractor.model.RunMetrics;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.Comparator;
import java.util.List;
import java.util.Set;
import java.util.stream.Collectors;
import java.util.stream.Stream;

public class DiscoveryStage {
    private final ExtractorConfig config;
    private final RunMetrics metrics;

    public DiscoveryStage(ExtractorConfig config, RunMetrics metrics) {
        this.config = config;
        this.metrics = metrics;
    }

    public List<Path> discover() throws IOException {
        if (!config.filesFilter().isEmpty()) {
            List<Path> files = discoverFromFilter();
            metrics.filesDiscovered.set(files.size());
            return files;
        }
        try (Stream<Path> stream = Files.walk(config.repoPath())) {
            List<Path> files = stream
                    .filter(Files::isRegularFile)
                    .filter(path -> path.toString().endsWith(".java"))
                    .filter(path -> !isExcluded(path))
                    .filter(path -> !config.skipTests() || !isTestPath(path))
                    .sorted(Comparator.comparing(path -> config.repoPath().relativize(path).toString().replace('\\', '/')))
                    .collect(Collectors.toList());
            metrics.filesDiscovered.set(files.size());
            return files;
        }
    }

    private List<Path> discoverFromFilter() {
        Set<Path> uniquePaths = config.filesFilter().stream()
                .map(path -> path.isAbsolute() ? path.normalize() : config.repoPath().resolve(path).normalize())
                .filter(path -> path.startsWith(config.repoPath()))
                .filter(path -> path.toString().endsWith(".java"))
                .filter(Files::isRegularFile)
                .filter(path -> !isExcluded(path))
                .filter(path -> !config.skipTests() || !isTestPath(path))
                .collect(Collectors.toSet());
        return uniquePaths.stream()
                .sorted(Comparator.comparing(path -> config.repoPath().relativize(path).toString().replace('\\', '/')))
                .collect(Collectors.toList());
    }

    private boolean isExcluded(Path path) {
        String normalized = config.repoPath().relativize(path).toString().replace('\\', '/');
        return normalized.contains("/target/")
                || normalized.startsWith("target/")
                || normalized.contains("/generated-sources/")
                || normalized.startsWith("generated-sources/")
                || normalized.contains("/generated/")
                || normalized.startsWith("generated/")
                || normalized.contains("/generated-test-sources/")
                || normalized.startsWith("generated-test-sources/")
                || normalized.startsWith(".mvn/")
                || normalized.contains("/.mvn/");
    }

    private boolean isTestPath(Path path) {
        String normalized = config.repoPath().relativize(path).toString().replace('\\', '/');
        return normalized.contains("/src/test/") || normalized.startsWith("src/test/");
    }
}
