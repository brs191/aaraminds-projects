# RIF Phase 1 Extractor — Known Issues and Advisories

**Extractor version:** 1.0.0-SNAPSHOT  
**Target repo:** `apm0045942-credit-routing-service` (Java 17 / Spring Boot)  
**Last reviewed:** 2026-06-10

---

## KI-001 — `same_file_resolution_failure_count` is dominated by `@Slf4j` false-positives

### Symptom

Full-repo run against `apm0045942-credit-routing-service` reports:

```
same_file_resolution_failure_count : 14265
```

This is a **94.8% failure rate** (14,265 fail / 781 pass) and will look alarming in CI if
interpreted without context.

### Root cause

The repo uses Lombok's `@Slf4j` annotation on almost every class:

```java
@Slf4j
public class CCRoutingService {
    // Lombok generates: private static final Logger log = LoggerFactory.getLogger(CCRoutingService.class);
    ...
    log.info("routing to CC API");  // ← every call to log.xxx() triggers a resolution attempt
}
```

Lombok generates the `log` field at compile time via annotation processing. JavaParser sees
only the source AST — the `log` field **has no AST node** in the `.java` file. When the
`CallVisitor` visits `log.info(...)`, it attempts to resolve the callee via SymbolSolver, fails
(no AST node to resolve against), and increments `same_file_resolution_failure_count`.

### Why this is NOT a data quality problem

1. `log.xxx()` calls target the **SLF4J JAR** (`org.slf4j.Logger`), not a method declared in
   the same file. They would never produce `SAME_FILE_CALLS` edges even if resolved.
2. The graph is **complete**: no `SAME_FILE_CALLS` edges are silently dropped due to this
   failure. The failures account for zero missing edges.
3. The metric counts resolution attempts, not dropped edges.

### CI gate behaviour

The Step 1.8 CI gate checks `unsupported_construct_count` (must be 0 on first-party files).
It does **not** fail on `same_file_resolution_failure_count` — this metric is advisory only.

If a team adds a threshold on `same_file_resolution_failure_count`, use a **per-class
normalised rate** rather than an absolute count, and exclude classes annotated with `@Slf4j`,
`@Log4j2`, `@CommonsLog`, or `@XSlf4j`.

### Baseline (for CI advisory threshold)

| Metric | Value | Repo |
|--------|-------|------|
| `same_file_resolution_failure_count` | 14,265 | `apm0045942-credit-routing-service` @ `44b6b865` |
| `same_file_calls_count` | 781 | same |
| Failure rate | 94.8% | dominated by `@Slf4j` |
| Root cause | `@Slf4j log.xxx()` calls — Lombok field, no AST node | |

### Possible Phase 2 improvement

In Phase 2, when Lombok field injection is tracked (`INJECTS` edges, `provenance_kind=generated`),
the extractor could pre-register `log` as an external field and skip resolution attempts
on `log.xxx()` calls. This would bring the failure rate close to 0% without any data loss.

---

## KI-002 — `CompactConstructorDeclaration` and `EnumDeclaration` fallbacks (resolved)

**Status:** ✅ Resolved in `CallVisitor.java` — `unsupported_construct_count = 0`

### What was happening

JavaParser 3.24+ represents a Record's compact constructor as `CompactConstructorDeclaration`,
which is a **sibling** of `ConstructorDeclaration` in the AST hierarchy — not a subtype.
`instanceof ConstructorDeclaration` returns `false` for compact constructors.

Similarly, `EnumDeclaration`'s synthetic `values()` / `valueOf()` methods — when resolved
via SymbolSolver and `.toAst()` called — return the `EnumDeclaration` node itself, not a
`MethodDeclaration`.

Both caused `calleeNodeId()` to fall through to the unsupported-construct counter, resulting
in 4 garbage placeholder edges.

### Fix applied

`CallVisitor.calleeNodeId()` now handles three cases:

1. `MethodDeclaration` — standard method (unchanged)
2. `ConstructorDeclaration` — standard class constructor (unchanged)
3. `CompactConstructorDeclaration` — Record compact constructor: uses params from the
   enclosing `RecordDeclaration.getParameters()` and emits a `CONSTRUCTOR` node id
4. `EnumDeclaration` — silent skip (synthetic methods have no user-authored AST node;
   they would never produce meaningful `SAME_FILE_CALLS` edges)

Post-fix: `unsupported_construct_count = 0`, 4 fewer garbage edges, byte-identical output.
