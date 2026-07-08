package com.aaraminds.rif.extractor.resolve;

import com.github.javaparser.Position;
import com.github.javaparser.ast.CompilationUnit;
import com.github.javaparser.ast.Node;
import com.github.javaparser.ast.body.TypeDeclaration;
import com.github.javaparser.ast.expr.ObjectCreationExpr;
import java.util.ArrayDeque;
import java.util.ArrayList;
import java.util.Deque;
import java.util.List;
import java.util.Optional;

public final class QualifiedNameUtils {
    private QualifiedNameUtils() {
    }

    public static String packageName(CompilationUnit compilationUnit) {
        return compilationUnit.getPackageDeclaration().map(pd -> pd.getNameAsString()).orElse("");
    }

    public static String binaryName(TypeDeclaration<?> typeDeclaration, CompilationUnit compilationUnit) {
        String packageName = packageName(compilationUnit);
        List<String> names = typeNamePath(typeDeclaration);
        String joined = String.join("$", names);
        return packageName.isEmpty() ? joined : packageName + "." + joined;
    }

    public static String canonicalName(TypeDeclaration<?> typeDeclaration, CompilationUnit compilationUnit) {
        String packageName = packageName(compilationUnit);
        List<String> names = typeNamePath(typeDeclaration);
        String joined = String.join(".", names);
        return packageName.isEmpty() ? joined : packageName + "." + joined;
    }

    public static String anonymousBinaryName(ObjectCreationExpr objectCreationExpr, String enclosingBinaryName) {
        Optional<Position> begin = objectCreationExpr.getBegin();
        int line = begin.map(position -> position.line).orElse(0);
        int column = begin.map(position -> position.column).orElse(0);
        return enclosingBinaryName + "$anon_" + line + "_" + column;
    }

    public static TypeDeclaration<?> enclosingType(Node node) {
        Node current = node;
        while (current != null) {
            if (current instanceof TypeDeclaration<?> typeDeclaration) {
                return typeDeclaration;
            }
            current = current.getParentNode().orElse(null);
        }
        throw new IllegalStateException("No enclosing type declaration found");
    }

    private static List<String> typeNamePath(TypeDeclaration<?> typeDeclaration) {
        Deque<String> names = new ArrayDeque<>();
        Node current = typeDeclaration;
        while (current != null) {
            if (current instanceof TypeDeclaration<?> currentType) {
                names.addFirst(currentType.getNameAsString());
            }
            current = current.getParentNode().orElse(null);
        }
        return new ArrayList<>(names);
    }
}
