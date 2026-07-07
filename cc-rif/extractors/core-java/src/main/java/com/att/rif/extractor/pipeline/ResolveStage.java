package com.att.rif.extractor.pipeline;

import com.att.rif.extractor.ExtractorConfig;
import com.att.rif.extractor.model.RunMetrics;
import com.att.rif.extractor.model.StubNode;
import com.att.rif.extractor.resolve.QualifiedNameUtils;
import com.att.rif.extractor.resolve.TypeMetadata;
import com.att.rif.extractor.visitor.CallVisitor;
import com.att.rif.extractor.visitor.FileVisitor;
import com.att.rif.extractor.visitor.MemberVisitor;
import com.att.rif.extractor.visitor.TypeVisitor;
import com.github.javaparser.JavaParser;
import com.github.javaparser.ParseResult;
import com.github.javaparser.ParserConfiguration;
import com.github.javaparser.ast.CompilationUnit;
import com.github.javaparser.ast.body.ClassOrInterfaceDeclaration;
import com.github.javaparser.ast.body.EnumDeclaration;
import com.github.javaparser.ast.body.RecordDeclaration;
import com.github.javaparser.ast.body.TypeDeclaration;
import java.io.IOException;
import java.nio.file.Path;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class ResolveStage {
    private static final Logger LOGGER = LoggerFactory.getLogger(ResolveStage.class);

    private final ExtractorConfig config;
    private final RunMetrics metrics;
    private final ParserConfiguration parserConfiguration;
    private final StubNode stubNodeRegistry = new StubNode();

    public ResolveStage(ExtractorConfig config, RunMetrics metrics, ParserConfiguration parserConfiguration) {
        this.config = config;
        this.metrics = metrics;
        this.parserConfiguration = parserConfiguration;
    }

    public ExtractionResult resolve(List<Path> files) throws IOException {
        Map<String, TypeMetadata> localTypes = buildLocalTypeIndex(files);
        JavaParser parser = new JavaParser(parserConfiguration);
        ExtractionResult result = new ExtractionResult();
        FileVisitor fileVisitor = new FileVisitor(config, metrics, stubNodeRegistry, localTypes);
        TypeVisitor typeVisitor = new TypeVisitor(config, metrics, stubNodeRegistry, localTypes);
        MemberVisitor memberVisitor = new MemberVisitor(config, metrics);
        CallVisitor callVisitor = new CallVisitor(config, metrics);

        for (Path file : files) {
            try {
                ParseResult<CompilationUnit> parseResult = parser.parse(file);
                if (parseResult.getResult().isEmpty()) {
                    metrics.filesFailed.incrementAndGet();
                    LOGGER.warn("Failed to parse {}", file);
                    continue;
                }
                CompilationUnit compilationUnit = parseResult.getResult().orElseThrow();
                metrics.filesParsed.incrementAndGet();
                String relativePath = relativePath(file);
                result.addNode(fileVisitor.visit(compilationUnit, file, relativePath));
                result.addEdges(fileVisitor.importEdges(compilationUnit, relativePath));
                TypeVisitor.TypeVisitResult typeVisitResult = typeVisitor.visit(compilationUnit, relativePath);
                result.addNodes(typeVisitResult.nodes());
                result.addEdges(typeVisitResult.edges());
                MemberVisitor.MemberVisitResult memberVisitResult = memberVisitor.visit(compilationUnit, relativePath, typeVisitResult.typeNodesByDeclaration());
                result.addNodes(memberVisitResult.nodes());
                result.addEdges(memberVisitResult.edges());
                result.addEdges(callVisitor.visit(compilationUnit, relativePath, typeVisitResult.typeNodesByDeclaration()));
            } catch (Exception exception) {
                metrics.filesFailed.incrementAndGet();
                LOGGER.warn("Failed processing file {}", file, exception);
            }
        }

        result.addNodes(stubNodeRegistry.all());
        return result;
    }

    private Map<String, TypeMetadata> buildLocalTypeIndex(List<Path> files) throws IOException {
        Map<String, TypeMetadata> localTypes = new HashMap<>();
        JavaParser parser = new JavaParser(parserConfiguration);
        for (Path file : files) {
            ParseResult<CompilationUnit> parseResult = parser.parse(file);
            if (parseResult.getResult().isEmpty()) {
                continue;
            }
            CompilationUnit compilationUnit = parseResult.getResult().orElseThrow();
            for (TypeDeclaration<?> typeDeclaration : compilationUnit.findAll(TypeDeclaration.class)) {
                TypeMetadata metadata = new TypeMetadata(
                        QualifiedNameUtils.binaryName(typeDeclaration, compilationUnit),
                        QualifiedNameUtils.canonicalName(typeDeclaration, compilationUnit),
                        kindOf(typeDeclaration));
                localTypes.put(metadata.binaryName(), metadata);
                localTypes.put(metadata.canonicalName(), metadata);
            }
        }
        return localTypes;
    }

    private String kindOf(TypeDeclaration<?> typeDeclaration) {
        if (typeDeclaration instanceof RecordDeclaration) {
            return "RECORD";
        }
        if (typeDeclaration instanceof EnumDeclaration) {
            return "ENUM";
        }
        if (typeDeclaration instanceof ClassOrInterfaceDeclaration declaration) {
            return declaration.isInterface() ? "INTERFACE" : "CLASS";
        }
        return "CLASS";
    }

    private String relativePath(Path file) {
        return config.repoPath().relativize(file).toString().replace('\\', '/');
    }
}
