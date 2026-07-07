package com.att.rif.extractor.visitor;

import com.att.rif.extractor.ExtractorConfig;
import com.att.rif.extractor.model.EdgeRecord;
import com.att.rif.extractor.model.NodeRecord;
import com.att.rif.extractor.model.RunMetrics;
import com.att.rif.extractor.resolve.NodeIdComputer;
import com.att.rif.extractor.resolve.SourceRefBuilder;
import com.att.rif.extractor.resolve.TypeResolver;
import com.github.javaparser.Position;
import com.github.javaparser.ast.CompilationUnit;
import com.github.javaparser.ast.body.BodyDeclaration;
import com.github.javaparser.ast.body.ConstructorDeclaration;
import com.github.javaparser.ast.body.FieldDeclaration;
import com.github.javaparser.ast.body.MethodDeclaration;
import com.github.javaparser.ast.body.Parameter;
import com.github.javaparser.ast.body.TypeDeclaration;
import com.github.javaparser.ast.expr.AnnotationExpr;
import com.github.javaparser.ast.nodeTypes.modifiers.NodeWithAccessModifiers;
import com.github.javaparser.ast.type.Type;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class MemberVisitor {
    private final ExtractorConfig config;
    private final RunMetrics metrics;
    private final TypeResolver typeResolver = new TypeResolver();

    public MemberVisitor(ExtractorConfig config, RunMetrics metrics) {
        this.config = config;
        this.metrics = metrics;
    }

    public MemberVisitResult visit(
            CompilationUnit compilationUnit,
            String relativePath,
            Map<TypeDeclaration<?>, TypeVisitor.TypeNodeContext> typeNodesByDeclaration) {
        List<Map<String, Object>> nodes = new ArrayList<>();
        List<Map<String, Object>> edges = new ArrayList<>();
        Map<BodyDeclaration<?>, String> callableNodeIds = new HashMap<>();

        for (Map.Entry<TypeDeclaration<?>, TypeVisitor.TypeNodeContext> entry : typeNodesByDeclaration.entrySet()) {
            TypeDeclaration<?> typeDeclaration = entry.getKey();
            TypeVisitor.TypeNodeContext typeContext = entry.getValue();
            for (BodyDeclaration<?> member : typeDeclaration.getMembers()) {
                if (member instanceof MethodDeclaration methodDeclaration) {
                    List<String> paramTypes = parameterTypes(methodDeclaration.getParameters());
                    String qualifiedName = methodQualifiedName(typeContext.binaryName(), methodDeclaration.getNameAsString(), paramTypes);
                    Map<String, Object> node = NodeRecord.methodNode(
                            config.repoId(),
                            qualifiedName,
                            "METHOD",
                            tokenSourceRef(methodDeclaration.getType().getBegin().map(position -> position.line).orElse(null), relativePath),
                            methodDeclaration.getNameAsString(),
                            typeResolver.resolveTypeName(methodDeclaration.getType(), metrics),
                            paramTypes,
                            methodDeclaration.isStatic(),
                            visibility(methodDeclaration),
                            annotations(methodDeclaration.getAnnotations()));
                    nodes.add(node);
                    callableNodeIds.put(methodDeclaration, String.valueOf(node.get("node_id")));
                } else if (member instanceof ConstructorDeclaration constructorDeclaration) {
                    List<String> paramTypes = parameterTypes(constructorDeclaration.getParameters());
                    String qualifiedName = constructorQualifiedName(typeContext.binaryName(), paramTypes);
                    Map<String, Object> node = NodeRecord.methodNode(
                            config.repoId(),
                            qualifiedName,
                            "CONSTRUCTOR",
                            tokenSourceRef(constructorDeclaration.getName().getBegin().map(position -> position.line).orElse(null), relativePath),
                            constructorDeclaration.getNameAsString(),
                            null,
                            paramTypes,
                            false,
                            visibility(constructorDeclaration),
                            annotations(constructorDeclaration.getAnnotations()));
                    nodes.add(node);
                    callableNodeIds.put(constructorDeclaration, String.valueOf(node.get("node_id")));
                } else if (member instanceof FieldDeclaration fieldDeclaration) {
                    String sourceRef = tokenSourceRef(fieldDeclaration.getElementType().getBegin().map(position -> position.line).orElse(null), relativePath);
                    String typeName = typeResolver.resolveTypeName(fieldDeclaration.getElementType(), metrics);
                    for (var variable : fieldDeclaration.getVariables()) {
                        String qualifiedName = fieldQualifiedName(typeContext.binaryName(), variable.getNameAsString());
                        Map<String, Object> node = NodeRecord.fieldNode(
                                config.repoId(),
                                qualifiedName,
                                sourceRef,
                                variable.getNameAsString(),
                                typeName,
                                fieldDeclaration.isStatic(),
                                fieldDeclaration.isFinal(),
                                visibility(fieldDeclaration),
                                annotations(fieldDeclaration.getAnnotations()));
                        nodes.add(node);
                        String fieldNodeId = NodeIdComputer.computeNodeId(config.repoId(), qualifiedName, "FIELD");
                        edges.add(EdgeRecord.declaresField(typeContext.nodeId(), fieldNodeId, sourceRef));
                    }
                }
            }
        }

        return new MemberVisitResult(nodes, edges, callableNodeIds);
    }

    private List<String> parameterTypes(List<Parameter> parameters) {
        List<String> paramTypes = new ArrayList<>(parameters.size());
        for (Parameter parameter : parameters) {
            Type parameterType = parameter.getType();
            paramTypes.add(typeResolver.resolveParamTypeName(parameterType, metrics));
        }
        return paramTypes;
    }

    private List<String> annotations(List<AnnotationExpr> annotations) {
        return annotations.stream().map(annotationExpr -> annotationExpr.getName().getIdentifier()).sorted(Comparator.naturalOrder()).toList();
    }

    private String tokenSourceRef(Integer line, String relativePath) {
        if (line == null) {
            metrics.provenanceGapCount.incrementAndGet();
            return SourceRefBuilder.unavailable();
        }
        return SourceRefBuilder.build(config.repoId(), config.sha(), relativePath, line);
    }

    private String visibility(NodeWithAccessModifiers<?> declaration) {
        return switch (declaration.getAccessSpecifier()) {
            case PUBLIC -> "public";
            case PROTECTED -> "protected";
            case PRIVATE -> "private";
            case NONE -> "package_private";
        };
    }

    public static String methodQualifiedName(String classBinaryName, String methodName, List<String> paramTypes) {
        return classBinaryName + "#" + methodName + "(" + String.join(",", paramTypes) + ")";
    }

    public static String constructorQualifiedName(String classBinaryName, List<String> paramTypes) {
        return classBinaryName + "#<init>(" + String.join(",", paramTypes) + ")";
    }

    public static String fieldQualifiedName(String classBinaryName, String fieldName) {
        return classBinaryName + "#" + fieldName;
    }

    public record MemberVisitResult(
            List<Map<String, Object>> nodes,
            List<Map<String, Object>> edges,
            Map<BodyDeclaration<?>, String> callableNodeIds) {
    }
}
