package com.aaraminds.rif.extractor.resolve;

import com.github.javaparser.ast.CompilationUnit;
import com.github.javaparser.ast.ImportDeclaration;
import com.github.javaparser.ast.body.TypeDeclaration;
import java.util.Set;

public final class LombokDetector {
    private static final Set<String> LOMBOK_ANNOTATIONS = Set.of(
            "Data", "Value", "Builder", "Getter", "Setter", "RequiredArgsConstructor", "AllArgsConstructor",
            "NoArgsConstructor", "ToString", "EqualsAndHashCode", "Slf4j", "FieldDefaults", "SuperBuilder",
            "With", "Wither", "NonNull", "ConstructorProperties");

    private LombokDetector() {
    }

    public static boolean hasLombok(TypeDeclaration<?> typeDeclaration, CompilationUnit compilationUnit) {
        boolean imported = compilationUnit.getImports().stream()
                .map(ImportDeclaration::getNameAsString)
                .anyMatch(name -> name.startsWith("lombok."));
        boolean annotated = typeDeclaration.getAnnotations().stream()
                .map(annotationExpr -> annotationExpr.getName().getIdentifier())
                .anyMatch(LOMBOK_ANNOTATIONS::contains);
        return imported || annotated;
    }
}
