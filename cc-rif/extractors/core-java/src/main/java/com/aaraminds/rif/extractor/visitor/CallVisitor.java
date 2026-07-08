package com.aaraminds.rif.extractor.visitor;

import com.aaraminds.rif.extractor.ExtractorConfig;
import com.aaraminds.rif.extractor.model.EdgeRecord;
import com.aaraminds.rif.extractor.model.RunMetrics;
import com.aaraminds.rif.extractor.resolve.NodeIdComputer;
import com.aaraminds.rif.extractor.resolve.QualifiedNameUtils;
import com.aaraminds.rif.extractor.resolve.SourceRefBuilder;
import com.aaraminds.rif.extractor.resolve.TypeResolver;
import com.github.javaparser.Position;
import com.github.javaparser.ast.CompilationUnit;
import com.github.javaparser.ast.Node;
import com.github.javaparser.ast.body.BodyDeclaration;
import com.github.javaparser.ast.body.CompactConstructorDeclaration;
import com.github.javaparser.ast.body.ConstructorDeclaration;
import com.github.javaparser.ast.body.EnumDeclaration;
import com.github.javaparser.ast.body.MethodDeclaration;
import com.github.javaparser.ast.body.Parameter;
import com.github.javaparser.ast.body.RecordDeclaration;
import com.github.javaparser.ast.body.TypeDeclaration;
import com.github.javaparser.ast.expr.MethodCallExpr;
import com.github.javaparser.ast.expr.ObjectCreationExpr;
import com.github.javaparser.resolution.UnsolvedSymbolException;
import com.github.javaparser.resolution.declarations.ResolvedConstructorDeclaration;
import com.github.javaparser.resolution.declarations.ResolvedMethodDeclaration;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class CallVisitor {
    private static final Logger LOGGER = LoggerFactory.getLogger(CallVisitor.class);

    private final ExtractorConfig config;
    private final RunMetrics metrics;
    private final TypeResolver typeResolver = new TypeResolver();

    public CallVisitor(ExtractorConfig config, RunMetrics metrics) {
        this.config = config;
        this.metrics = metrics;
    }

    public List<Map<String, Object>> visit(
            CompilationUnit compilationUnit,
            String relativePath,
            Map<TypeDeclaration<?>, TypeVisitor.TypeNodeContext> typeNodesByDeclaration) {
        Map<String, Integer> minimumLines = new HashMap<>();
        for (MethodDeclaration methodDeclaration : compilationUnit.findAll(MethodDeclaration.class)) {
            TypeDeclaration<?> owner = QualifiedNameUtils.enclosingType(methodDeclaration);
            TypeVisitor.TypeNodeContext ownerContext = typeNodesByDeclaration.get(owner);
            if (ownerContext == null) {
                continue;
            }
            String callerNodeId = NodeIdComputer.computeNodeId(
                    config.repoId(),
                    MemberVisitor.methodQualifiedName(ownerContext.binaryName(), methodDeclaration.getNameAsString(), parameterTypes(methodDeclaration.getParameters(), true)),
                    "METHOD");
            collectMethodCalls(compilationUnit, minimumLines, callerNodeId, methodDeclaration);
            collectConstructorCalls(compilationUnit, minimumLines, callerNodeId, methodDeclaration);
        }

        for (ConstructorDeclaration constructorDeclaration : compilationUnit.findAll(ConstructorDeclaration.class)) {
            TypeDeclaration<?> owner = QualifiedNameUtils.enclosingType(constructorDeclaration);
            TypeVisitor.TypeNodeContext ownerContext = typeNodesByDeclaration.get(owner);
            if (ownerContext == null) {
                continue;
            }
            String callerNodeId = NodeIdComputer.computeNodeId(
                    config.repoId(),
                    MemberVisitor.constructorQualifiedName(ownerContext.binaryName(), parameterTypes(constructorDeclaration.getParameters(), true)),
                    "CONSTRUCTOR");
            collectMethodCalls(compilationUnit, minimumLines, callerNodeId, constructorDeclaration);
            collectConstructorCalls(compilationUnit, minimumLines, callerNodeId, constructorDeclaration);
        }

        List<Map<String, Object>> edges = new ArrayList<>();
        for (Map.Entry<String, Integer> entry : minimumLines.entrySet()) {
            String[] parts = entry.getKey().split("\\|", 2);
            edges.add(EdgeRecord.sameFileCalls(
                    parts[0],
                    parts[1],
                    SourceRefBuilder.build(config.repoId(), config.sha(), relativePath, entry.getValue())));
        }
        return edges;
    }

    private void collectMethodCalls(
            CompilationUnit compilationUnit,
            Map<String, Integer> minimumLines,
            String callerNodeId,
            BodyDeclaration<?> bodyDeclaration) {
        for (MethodCallExpr methodCallExpr : bodyDeclaration.findAll(MethodCallExpr.class)) {
            try {
                ResolvedMethodDeclaration resolvedMethod = methodCallExpr.resolve();
                resolvedMethod.toAst().ifPresent(calleeAst -> addIfSameCompilationUnit(
                        compilationUnit,
                        minimumLines,
                        callerNodeId,
                        calleeAst,
                        methodCallExpr,
                        "METHOD"));
            } catch (UnsolvedSymbolException exception) {
                metrics.sameFileResolutionFailureCount.incrementAndGet();
                LOGGER.debug("Failed resolving method call {}", methodCallExpr, exception);
            } catch (StackOverflowError error) {
                metrics.resolutionOverflowCount.incrementAndGet();
                LOGGER.warn("Overflow resolving method call {}", methodCallExpr);
            } catch (RuntimeException exception) {
                metrics.sameFileResolutionFailureCount.incrementAndGet();
                LOGGER.debug("Runtime failure resolving method call {}", methodCallExpr, exception);
            }
        }
    }

    private void collectConstructorCalls(
            CompilationUnit compilationUnit,
            Map<String, Integer> minimumLines,
            String callerNodeId,
            BodyDeclaration<?> bodyDeclaration) {
        for (ObjectCreationExpr objectCreationExpr : bodyDeclaration.findAll(ObjectCreationExpr.class)) {
            try {
                ResolvedConstructorDeclaration resolvedConstructor = objectCreationExpr.resolve();
                resolvedConstructor.toAst().ifPresent(calleeAst -> addIfSameCompilationUnit(
                        compilationUnit,
                        minimumLines,
                        callerNodeId,
                        calleeAst,
                        objectCreationExpr,
                        "CONSTRUCTOR"));
            } catch (UnsolvedSymbolException exception) {
                metrics.sameFileResolutionFailureCount.incrementAndGet();
                LOGGER.debug("Failed resolving constructor call {}", objectCreationExpr, exception);
            } catch (StackOverflowError error) {
                metrics.resolutionOverflowCount.incrementAndGet();
                LOGGER.warn("Overflow resolving constructor call {}", objectCreationExpr);
            } catch (RuntimeException exception) {
                metrics.sameFileResolutionFailureCount.incrementAndGet();
                LOGGER.debug("Runtime failure resolving constructor call {}", objectCreationExpr, exception);
            }
        }
    }

    private void addIfSameCompilationUnit(
            CompilationUnit compilationUnit,
            Map<String, Integer> minimumLines,
            String callerNodeId,
            Node calleeAst,
            Node callSite,
            String callableKind) {
        CompilationUnit calleeCompilationUnit = calleeAst.findCompilationUnit().orElse(null);
        if (calleeCompilationUnit != compilationUnit) {
            return;
        }
        String calleeNodeId = calleeNodeId(calleeAst, callableKind);
        if (calleeNodeId == null) {
            return; // unsupported node type — already logged and counted in calleeNodeId()
        }
        int line = callSite.getBegin().map(position -> position.line).orElseGet(() -> {
            metrics.provenanceGapCount.incrementAndGet();
            return 1;
        });
        String key = callerNodeId + "|" + calleeNodeId;
        minimumLines.merge(key, line, Math::min);
    }

    private String calleeNodeId(Node calleeAst, String callableKind) {
        if (calleeAst instanceof MethodDeclaration methodDeclaration) {
            TypeDeclaration<?> owner = QualifiedNameUtils.enclosingType(methodDeclaration);
            String ownerBinary = QualifiedNameUtils.binaryName(owner, methodDeclaration.findCompilationUnit().orElseThrow());
            return NodeIdComputer.computeNodeId(
                    config.repoId(),
                    MemberVisitor.methodQualifiedName(ownerBinary, methodDeclaration.getNameAsString(), parameterTypes(methodDeclaration.getParameters(), true)),
                    callableKind);
        }
        if (calleeAst instanceof ConstructorDeclaration constructorDeclaration) {
            TypeDeclaration<?> owner = QualifiedNameUtils.enclosingType(constructorDeclaration);
            String ownerBinary = QualifiedNameUtils.binaryName(owner, constructorDeclaration.findCompilationUnit().orElseThrow());
            return NodeIdComputer.computeNodeId(
                    config.repoId(),
                    MemberVisitor.constructorQualifiedName(ownerBinary, parameterTypes(constructorDeclaration.getParameters(), true)),
                    callableKind);
        }
        // Java 16+ Record compact constructor — CompactConstructorDeclaration is a sibling of
        // ConstructorDeclaration, NOT a subtype. Must be handled explicitly.
        // Parameters are NOT on the CompactConstructorDeclaration — they live on the
        // enclosing RecordDeclaration as record components.
        if (calleeAst instanceof CompactConstructorDeclaration compactConstructor) {
            TypeDeclaration<?> owner = QualifiedNameUtils.enclosingType(compactConstructor);
            String ownerBinary = QualifiedNameUtils.binaryName(owner, compactConstructor.findCompilationUnit().orElseThrow());
            List<String> paramTypes = (owner instanceof RecordDeclaration rec)
                    ? parameterTypes(rec.getParameters(), true)
                    : List.of();
            return NodeIdComputer.computeNodeId(
                    config.repoId(),
                    MemberVisitor.constructorQualifiedName(ownerBinary, paramTypes),
                    "CONSTRUCTOR");
        }
        // Enum synthetic methods (values(), valueOf()) — SymbolSolver maps toAst() back to the
        // EnumDeclaration node itself. Not meaningful SAME_FILE_CALLS targets; skip silently.
        if (calleeAst instanceof EnumDeclaration) {
            return null;
        }
        LOGGER.warn("Unhandled callee AST node type {} at {}; skipping SAME_FILE_CALLS edge",
                calleeAst.getClass().getSimpleName(),
                calleeAst.getBegin().map(Object::toString).orElse("unknown"));
        metrics.unsupportedConstructCount.incrementAndGet();
        return null;
    }

    private List<String> parameterTypes(List<Parameter> parameters, boolean parameterContext) {
        List<String> types = new ArrayList<>(parameters.size());
        for (Parameter parameter : parameters) {
            types.add(parameterContext
                    ? typeResolver.resolveParamTypeName(parameter.getType(), metrics)
                    : typeResolver.resolveTypeName(parameter.getType(), metrics));
        }
        return types;
    }
}
