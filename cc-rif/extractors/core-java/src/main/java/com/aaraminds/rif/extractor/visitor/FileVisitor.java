package com.aaraminds.rif.extractor.visitor;

import com.aaraminds.rif.extractor.ExtractorConfig;
import com.aaraminds.rif.extractor.model.EdgeRecord;
import com.aaraminds.rif.extractor.model.NodeRecord;
import com.aaraminds.rif.extractor.model.RunMetrics;
import com.aaraminds.rif.extractor.model.StubNode;
import com.aaraminds.rif.extractor.resolve.NodeIdComputer;
import com.aaraminds.rif.extractor.resolve.SourceRefBuilder;
import com.aaraminds.rif.extractor.resolve.TypeMetadata;
import com.github.javaparser.Position;
import com.github.javaparser.ast.CompilationUnit;
import com.github.javaparser.ast.ImportDeclaration;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;

public class FileVisitor {
    private final ExtractorConfig config;
    private final RunMetrics metrics;
    private final StubNode stubNodeRegistry;
    private final Map<String, TypeMetadata> localTypes;

    public FileVisitor(ExtractorConfig config, RunMetrics metrics, StubNode stubNodeRegistry, Map<String, TypeMetadata> localTypes) {
        this.config = config;
        this.metrics = metrics;
        this.stubNodeRegistry = stubNodeRegistry;
        this.localTypes = localTypes;
    }

    public Map<String, Object> visit(CompilationUnit compilationUnit, Path file, String relativePath) throws IOException {
        int lineCount = Files.readAllLines(file, StandardCharsets.UTF_8).size();
        return NodeRecord.fileNode(
                config.repoId(),
                relativePath,
                SourceRefBuilder.build(config.repoId(), config.sha(), relativePath, 1),
                compilationUnit.getPackageDeclaration().map(pd -> pd.getNameAsString()).orElse(null),
                lineCount);
    }

    public List<Map<String, Object>> importEdges(CompilationUnit compilationUnit, String relativePath) {
        List<Map<String, Object>> edges = new ArrayList<>();
        String fileNodeId = NodeIdComputer.computeNodeId(config.repoId(), relativePath, "FILE");
        for (ImportDeclaration importDeclaration : compilationUnit.getImports()) {
            if (importDeclaration.isAsterisk() || importDeclaration.isStatic()) {
                continue;
            }
            String importedName = importDeclaration.getNameAsString();
            TypeMetadata metadata = localTypes.get(importedName);
            String toNodeId;
            if (metadata != null) {
                toNodeId = NodeIdComputer.computeNodeId(config.repoId(), metadata.binaryName(), metadata.kind());
            } else {
                stubNodeRegistry.getOrCreate(config.repoId(), importedName, "CLASS");
                toNodeId = NodeIdComputer.computeNodeId(config.repoId(), importedName, "CLASS");
            }
            // G8: Use UNAVAILABLE instead of fallback to line 1 when position unavailable
            String sourceRef = importDeclaration.getBegin()
                    .map(position -> SourceRefBuilder.build(config.repoId(), config.sha(), relativePath, position.line))
                    .orElseGet(() -> {
                        metrics.provenanceGapCount.incrementAndGet();
                        return SourceRefBuilder.unavailable();
                    });
            edges.add(EdgeRecord.imports(fileNodeId, toNodeId, sourceRef));
        }
        return edges;
    }
}
