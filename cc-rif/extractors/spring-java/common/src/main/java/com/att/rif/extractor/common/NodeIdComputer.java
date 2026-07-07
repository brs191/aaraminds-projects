package com.att.rif.extractor.common;

import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.HexFormat;

/**
 * Content-addressed node and edge ID computation.
 * <p>
 * All IDs are SHA-256 hex strings. Separator is NUL — matches the core extractor
 * {@code extractors/core-java/.../resolve/NodeIdComputer.java} exactly.
 * <p>
 * Special-purpose IDs (APPLICATION_CONTEXT, POINTCUT_EXPR, URL_ENDPOINT) use their own
 * prefix strings rather than the normal repoId+qualified_name+kind triple, because they
 * are synthetic nodes with no first-party source declaration.
 */
public final class NodeIdComputer {

    private NodeIdComputer() {
    }

    /** Normal node ID: SHA-256(repoId NUL qualifiedName NUL kind). Matches core-java. */
    public static String computeNodeId(String repoId, String qualifiedName, String kind) {
        return sha256(repoId + "\u0000" + qualifiedName + "\u0000" + kind);
    }

    /** Edge ID: SHA-256(fromNodeId NUL label NUL toNodeId). Matches core-java. */
    public static String computeEdgeId(String fromNodeId, String label, String toNodeId) {
        return sha256(fromNodeId + "\u0000" + label + "\u0000" + toNodeId);
    }

    /**
     * APPLICATION_CONTEXT virtual node ID: SHA-256("APPLICATION_CONTEXT:{repoId}").
     * Colon separator distinguishes it from normal code nodes (§1.2 CODE_MODEL).
     */
    public static String applicationContextNodeId(String repoId) {
        return sha256("APPLICATION_CONTEXT:" + repoId);
    }

    /**
     * POINTCUT_EXPRESSION synthetic node ID:
     * SHA-256("POINTCUT_EXPR:{repoId}:{aspectClassFqn}:{adviceMethodName}:{line}").
     */
    public static String pointcutExprNodeId(String repoId, String aspectClassFqn,
                                             String adviceMethodName, int line) {
        return sha256("POINTCUT_EXPR:" + repoId + ":" + aspectClassFqn + ":" + adviceMethodName + ":" + line);
    }

    /**
     * URL_ENDPOINT synthetic node ID:
     * SHA-256("URL_ENDPOINT:{repoId}:{callingMethodFqn}:{line}").
     */
    public static String urlEndpointNodeId(String repoId, String callingMethodFqn, int line) {
        return sha256("URL_ENDPOINT:" + repoId + ":" + callingMethodFqn + ":" + line);
    }

    /** Raw SHA-256 hex of a UTF-8 encoded string. Public for tests. */
    public static String sha256(String value) {
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            return HexFormat.of().formatHex(digest.digest(value.getBytes(StandardCharsets.UTF_8)));
        } catch (NoSuchAlgorithmException e) {
            throw new IllegalStateException("SHA-256 unavailable", e);
        }
    }
}
