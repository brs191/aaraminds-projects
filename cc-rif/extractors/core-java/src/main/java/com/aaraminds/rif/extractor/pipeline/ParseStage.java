package com.aaraminds.rif.extractor.pipeline;

import com.aaraminds.rif.extractor.ExtractorConfig;
import com.aaraminds.rif.extractor.model.RunMetrics;
import com.github.javaparser.ParserConfiguration;
import com.github.javaparser.symbolsolver.JavaSymbolSolver;
import com.github.javaparser.symbolsolver.resolution.typesolvers.CombinedTypeSolver;
import com.github.javaparser.symbolsolver.resolution.typesolvers.JarTypeSolver;
import com.github.javaparser.symbolsolver.resolution.typesolvers.JavaParserTypeSolver;
import com.github.javaparser.symbolsolver.resolution.typesolvers.ReflectionTypeSolver;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.List;
import java.util.stream.Collectors;
import java.util.stream.Stream;

public class ParseStage {
    private final ExtractorConfig config;
    @SuppressWarnings("unused")
    private final RunMetrics metrics;

    public ParseStage(ExtractorConfig config, RunMetrics metrics) {
        this.config = config;
        this.metrics = metrics;
    }

    public ParserConfiguration buildParserConfiguration() throws IOException {
        CombinedTypeSolver combinedTypeSolver = new CombinedTypeSolver();
        combinedTypeSolver.add(new ReflectionTypeSolver(false));
        for (Path sourceRoot : detectSourceRoots()) {
            combinedTypeSolver.add(new JavaParserTypeSolver(sourceRoot));
        }
        if (config.depsPath() != null && Files.exists(config.depsPath())) {
            try (Stream<Path> stream = Files.walk(config.depsPath())) {
                List<Path> jars = stream
                        .filter(Files::isRegularFile)
                        .filter(path -> path.toString().endsWith(".jar"))
                        .sorted(Comparator.comparing(Path::toString))
                        .collect(Collectors.toList());
                for (Path jar : jars) {
                    combinedTypeSolver.add(new JarTypeSolver(jar));
                }
            }
        }
        return new ParserConfiguration()
                .setLanguageLevel(ParserConfiguration.LanguageLevel.JAVA_17)
                .setStoreTokens(true)
                .setSymbolResolver(new JavaSymbolSolver(combinedTypeSolver));
    }

    private List<Path> detectSourceRoots() {
        List<Path> roots = new ArrayList<>();
        Path mainJava = config.repoPath().resolve("src/main/java");
        Path testJava = config.repoPath().resolve("src/test/java");
        if (Files.isDirectory(mainJava)) {
            roots.add(mainJava);
        }
        if (!config.skipTests() && Files.isDirectory(testJava)) {
            roots.add(testJava);
        }
        if (roots.isEmpty()) {
            roots.add(config.repoPath());
        }
        return roots;
    }
}
