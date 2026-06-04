package com.aaraminds.repointel;

/** Deterministic identity scheme (M0 SCHEMA.md). Natural keys are overload-aware
 *  FQNs — never a line number, never a sequence id — so the same SHA yields
 *  byte-identical ids and MERGE-on-id is idempotent. */
public final class IdGen {
    private IdGen() {}
    public static String type(String fqn)                                { return "type:" + fqn; }
    public static String method(String ownerFqn, String name, String ps) { return "method:" + ownerFqn + "#" + name + "(" + ps + ")"; }
    public static String field(String ownerFqn, String name)             { return "field:" + ownerFqn + "#" + name; }
    public static String endpoint(String httpMethod, String path)        { return "endpoint:" + httpMethod + " " + path; }
    public static String edge(String type, String src, String dst)       { return "edge:" + type + ":" + src + "->" + dst; }
}
