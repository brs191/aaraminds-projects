package com.aaraminds.rif.extractor.visitor;

import com.aaraminds.rif.extractor.ExtractorConfig;
import com.aaraminds.rif.extractor.model.EdgeRecord;
import com.aaraminds.rif.extractor.model.NodeRecord;
import com.aaraminds.rif.extractor.model.RunMetrics;
import com.aaraminds.rif.extractor.model.StubNode;
import com.aaraminds.rif.extractor.resolve.LombokDetector;
import com.aaraminds.rif.extractor.resolve.NodeIdComputer;
import com.aaraminds.rif.extractor.resolve.QualifiedNameUtils;
import com.aaraminds.rif.extractor.resolve.SourceRefBuilder;
import com.aaraminds.rif.extractor.resolve.TypeMetadata;
import com.github.javaparser.Position;
import com.github.javaparser.ast.CompilationUnit;
import com.github.javaparser.ast.body.ClassOrInterfaceDeclaration;
import com.github.javaparser.ast.body.EnumDeclaration;
import com.github.javaparser.ast.body.RecordDeclaration;
import com.github.javaparser.ast.body.TypeDeclaration;
import com.github.javaparser.ast.expr.AnnotationExpr;
import com.github.javaparser.ast.type.ClassOrInterfaceType;
import com.github.javaparser.resolution.UnsolvedSymbolException;
import com.github.javaparser.resolution.declarations.ResolvedReferenceTypeDeclaration;
import com.github.javaparser.resolution.types.ResolvedType;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class TypeVisitor {
    private static final Logger LOGGER = LoggerFactory.getLogger(TypeVisitor.class);

    private final ExtractorConfig config;
    private final RunMetrics metrics;
    private final StubNode stubNodeRegistry;
    private final Map<String, TypeMetadata> localTypes;

    public TypeVisitor(ExtractorConfig config, RunMetrics metrics, StubNode stubNodeRegistry, Map<String, TypeMetadata> localTypes) {
        this.config = config;
        this.metrics = metrics;
        this.stubNodeRegistry = stubNodeRegistry;
        this.localTypes = localTypes;
    }

    public TypeVisitResult visit(CompilationUnit compilationUnit, String relativePath) {
        List<Map<String, Object>> nodes = new ArrayList<>();
        List<Map<String, Object>> edges = new ArrayList<>();
        Map<TypeDeclaration<?>, TypeNodeContext> typeNodes = new HashMap<>();

        for (ClassOrInterfaceDeclaration declaration : compilationUnit.findAll(ClassOrInterfaceDeclaration.class)) {
            String kind = declaration.isInterface() ? "INTERFACE" : "CLASS";
            String binaryName = QualifiedNameUtils.binaryName(declaration, compilationUnit);
            String sourceRef = typeSourceRef(declaration, relativePath);
            List<String> annotations = sortedAnnotations(declaration.getAnnotations());
            boolean lombokPresent = LombokDetector.hasLombok(declaration, compilationUnit);
            Map<String, Object> node = declaration.isInterface()
                    ? NodeRecord.interfaceNode(config.repoId(), binaryName, sourceRef, declaration.getNameAsString(), !declaration.isTopLevelType(), annotations, lombokPresent, permitsTypes(declaration), "first_party", "file", "exact")
                    : NodeRecord.classNode(config.repoId(), binaryName, kind, sourceRef, declaration.getNameAsString(), declaration.isAbstract(), !declaration.isTopLevelType(), annotations, lombokPresent, permitsTypes(declaration), "first_party", "file", "exact");
            nodes.add(node);
            typeNodes.put(declaration, new TypeNodeContext(binaryName, kind, String.valueOf(node.get("node_id"))));
            addInheritanceEdges(declaration, relativePath, edges, typeNodes.get(declaration));
        }

        for (EnumDeclaration declaration : compilationUnit.findAll(EnumDeclaration.class)) {
            String binaryName = QualifiedNameUtils.binaryName(declaration, compilationUnit);
            Map<String, Object> node = NodeRecord.enumNode(
                    config.repoId(),
                    binaryName,
                    typeSourceRef(declaration, relativePath),
                    declaration.getNameAsString(),
                    !declaration.isTopLevelType(),
                    sortedAnnotations(declaration.getAnnotations()),
                    declaration.getEntries().stream().map(entry -> entry.getNameAsString()).sorted().toList(),
                    LombokDetector.hasLombok(declaration, compilationUnit),
                    "first_party",
                    "file",
                    "exact");
            nodes.add(node);
            typeNodes.put(declaration, new TypeNodeContext(binaryName, "ENUM", String.valueOf(node.get("node_id"))));
        }

        for (RecordDeclaration declaration : compilationUnit.findAll(RecordDeclaration.class)) {
            String binaryName = QualifiedNameUtils.binaryName(declaration, compilationUnit);
            Map<String, Object> node = NodeRecord.classNode(
                    config.repoId(),
                    binaryName,
                    "RECORD",
                    typeSourceRef(declaration, relativePath),
                    declaration.getNameAsString(),
                    false,
                    !declaration.isTopLevelType(),
                    sortedAnnotations(declaration.getAnnotations()),
                    LombokDetector.hasLombok(declaration, compilationUnit),
                    null,
                    "first_party",
                    "file",
                    "exact");
            nodes.add(node);
            typeNodes.put(declaration, new TypeNodeContext(binaryName, "RECORD", String.valueOf(node.get("node_id"))));
        }

        return new TypeVisitResult(nodes, edges, typeNodes);
    }

    private void addInheritanceEdges(
            ClassOrInterfaceDeclaration declaration,
            String relativePath,
            List<Map<String, Object>> edges,
            TypeNodeContext context) {
        String sourceRef = typeSourceRef(declaration, relativePath);
        for (ClassOrInterfaceType extendedType : declaration.getExtendedTypes()) {
            ResolvedTarget target = resolveTarget(extendedType, declaration, declaration.isInterface() ? "INTERFACE" : "CLASS");
            edges.add(EdgeRecord.extendsEdge(context.nodeId(), target.nodeId(), sourceRef));
        }
        if (!declaration.isInterface()) {
            for (ClassOrInterfaceType implementedType : declaration.getImplementedTypes()) {
                ResolvedTarget target = resolveTarget(implementedType, declaration, "INTERFACE");
                edges.add(EdgeRecord.implementsEdge(context.nodeId(), target.nodeId(), sourceRef));
            }
        }
    }

    private ResolvedTarget resolveTarget(ClassOrInterfaceType type, TypeDeclaration<?> owner, String fallbackKind) {
        try {
            ResolvedType resolvedType = type.resolve();
            ResolvedReferenceTypeDeclaration resolvedDeclaration = resolvedType.isReferenceType()
                    ? resolvedType.asReferenceType().getTypeDeclaration().orElse(null)
                    : null;
            if (resolvedDeclaration != null) {
                String qualifiedName = resolvedDeclaration.getQualifiedName();
                TypeMetadata metadata = localTypes.get(qualifiedName);
                String kind = resolvedDeclaration.isInterface() ? "INTERFACE" : resolvedDeclaration.isEnum() ? "ENUM" : "CLASS";
                if (metadata != null) {
                    return new ResolvedTarget(NodeIdComputer.computeNodeId(config.repoId(), metadata.binaryName(), metadata.kind()));
                }
                stubNodeRegistry.getOrCreate(config.repoId(), qualifiedName, kind);
                return new ResolvedTarget(NodeIdComputer.computeNodeId(config.repoId(), qualifiedName, kind));
            }
        } catch (UnsolvedSymbolException exception) {
            metrics.unresolvedTypeCount.incrementAndGet();
            LOGGER.debug("Unresolved inheritance target {}", type, exception);
        } catch (StackOverflowError error) {
            metrics.resolutionOverflowCount.incrementAndGet();
            LOGGER.warn("Overflow resolving inheritance target {}", type);
        } catch (RuntimeException exception) {
            metrics.unresolvedTypeCount.incrementAndGet();
            LOGGER.debug("Failed resolving inheritance target {}", type, exception);
        }

        String guessedCanonical = guessedCanonicalName(owner.findCompilationUnit().orElseThrow(), type);
        TypeMetadata metadata = localTypes.get(guessedCanonical);
        if (metadata != null) {
            return new ResolvedTarget(NodeIdComputer.computeNodeId(config.repoId(), metadata.binaryName(), metadata.kind()));
        }
        stubNodeRegistry.getOrCreate(config.repoId(), guessedCanonical, fallbackKind);
        return new ResolvedTarget(NodeIdComputer.computeNodeId(config.repoId(), guessedCanonical, fallbackKind));
    }

    private String guessedCanonicalName(CompilationUnit compilationUnit, ClassOrInterfaceType type) {
        String name = type.getNameWithScope();
        if (name.contains(".")) {
            return name;
        }
        String packageName = QualifiedNameUtils.packageName(compilationUnit);
        return packageName.isEmpty() ? name : packageName + "." + name;
    }

    private String typeSourceRef(TypeDeclaration<?> declaration, String relativePath) {
        return declaration.getName().getBegin()
                .map(position -> position.line)
                .map(line -> SourceRefBuilder.build(config.repoId(), config.sha(), relativePath, line))
                .orElseGet(() -> {
                    metrics.provenanceGapCount.incrementAndGet();
                    return SourceRefBuilder.unavailable();
                });
    }

    private List<String> sortedAnnotations(List<AnnotationExpr> annotations) {
        return annotations.stream()
                .map(annotationExpr -> annotationExpr.getName().getIdentifier())
                .sorted(Comparator.naturalOrder())
                .toList();
    }

    private List<String> permitsTypes(ClassOrInterfaceDeclaration declaration) {
        List<String> permits = declaration.getPermittedTypes().stream()
                .map(ClassOrInterfaceType::getNameWithScope)
                .sorted()
                .toList();
        return permits.isEmpty() ? null : permits;
    }

    public record TypeVisitResult(
            List<Map<String, Object>> nodes,
            List<Map<String, Object>> edges,
            Map<TypeDeclaration<?>, TypeNodeContext> typeNodesByDeclaration) {
    }

    public record TypeNodeContext(String binaryName, String kind, String nodeId) {
    }

    private record ResolvedTarget(String nodeId) {
    }
}
