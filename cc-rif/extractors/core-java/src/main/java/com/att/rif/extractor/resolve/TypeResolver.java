package com.att.rif.extractor.resolve;

import com.att.rif.extractor.model.RunMetrics;
import com.github.javaparser.ast.type.ArrayType;
import com.github.javaparser.ast.type.ClassOrInterfaceType;
import com.github.javaparser.ast.type.PrimitiveType;
import com.github.javaparser.ast.type.Type;
import com.github.javaparser.ast.type.VoidType;
import com.github.javaparser.resolution.UnsolvedSymbolException;
import com.github.javaparser.resolution.types.ResolvedType;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public class TypeResolver {
    private static final Logger LOGGER = LoggerFactory.getLogger(TypeResolver.class);

    public String resolveTypeName(Type type, RunMetrics metrics) {
        return resolve(type, metrics, false);
    }

    public String resolveParamTypeName(Type type, RunMetrics metrics) {
        return resolve(type, metrics, true);
    }

    private String resolve(Type type, RunMetrics metrics, boolean parameter) {
        if (type instanceof PrimitiveType primitiveType) {
            return primitiveType.asString();
        }
        if (type instanceof VoidType) {
            return "void";
        }
        if (type instanceof ArrayType arrayType) {
            return resolve(arrayType.getComponentType(), metrics, parameter) + "[]";
        }
        try {
            ResolvedType resolvedType = type.resolve();
            return describeResolvedType(resolvedType);
        } catch (UnsolvedSymbolException exception) {
            increment(metrics, parameter);
            LOGGER.debug("Unresolved type {}", type, exception);
            return fallback(type);
        } catch (StackOverflowError error) {
            metrics.resolutionOverflowCount.incrementAndGet();
            LOGGER.warn("Type resolution overflow for {}", type);
            return fallback(type);
        } catch (RuntimeException exception) {
            increment(metrics, parameter);
            LOGGER.debug("Failed to resolve type {}", type, exception);
            return fallback(type);
        }
    }

    private static String describeResolvedType(ResolvedType resolvedType) {
        if (resolvedType.isPrimitive()) {
            return resolvedType.asPrimitive().describe();
        }
        if (resolvedType.isVoid()) {
            return "void";
        }
        if (resolvedType.isArray()) {
            return describeResolvedType(resolvedType.asArrayType().getComponentType()) + "[]";
        }
        if (resolvedType.isReferenceType()) {
            return resolvedType.asReferenceType().getQualifiedName();
        }
        if (resolvedType.isTypeVariable() || resolvedType.isWildcard()) {
            return "java.lang.Object";
        }
        return resolvedType.describe() + "?";
    }

    private static void increment(RunMetrics metrics, boolean parameter) {
        if (parameter) {
            metrics.unresolvedParamTypeCount.incrementAndGet();
        } else {
            metrics.unresolvedTypeCount.incrementAndGet();
        }
    }

    private static String fallback(Type type) {
        if (type instanceof ClassOrInterfaceType classOrInterfaceType) {
            return classOrInterfaceType.getName().getIdentifier() + "?";
        }
        return type.asString() + "?";
    }
}
