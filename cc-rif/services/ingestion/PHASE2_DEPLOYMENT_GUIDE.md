# G1: Wiring Phase 2 Extractors into Production Ingestion

## Overview

Phase 2 extractors (DI, AOP, CrossService) emit Tier-B and Tier-C edges (INJECTS, PRODUCES, REGISTERS, ADVISES, CALLS_SOAP, CALLS_REST). They are **implemented and tested** but disabled by default in the production ingestion pipeline.

This document describes how to enable them.

## Architecture

The ingestion pipeline already supports Phase 2 extractors. The flow is:

```
1. Clone repo
2. Run Phase 1 Tier-A extractor (always)
   ↓
3. Run Phase 2 extractors (if enabled):
   - DI (Spring dependency injection)
   - AOP (Spring aspects / advice)
   - CrossService (SOAP/REST calls)
   ↓
4. Merge all NDJSON outputs into single stream
   ↓
5. Parse + validate (provenance gate)
   ↓
6. Bulk load into AGE graph
   ↓
7. Swap version pointer (atomic)
```

## Prerequisites

Before enabling Phase 2 in production:

1. ✅ Phase 1 schema applied (`phase-1/schema/age_schema.sql`, `phase-1/schema/relational_schema.sql`)
2. ✅ Phase 2 schema migrations applied:
   - `phase-2/schema/migration_phase2.sql` (Tier-B/C edge labels)
   - `phase-2/schema/migration_pgvector.sql` (vector embeddings)
   - `phase-2/schema/migration_fts.sql` (full-text search)
3. ✅ Phase 2 extractor JARs built and available:
   - `rif-extractor-phase2-di-shaded.jar`
   - `rif-extractor-phase2-aop-shaded.jar`
   - `rif-extractor-phase2-crossservice-shaded.jar`
4. ✅ Provenance gate extended to check edges: `phase-1/eval/provenance_check.py` (G6 — already done)
5. ✅ Ingestion service updated with G5 provenance gap detection

## Enabling Phase 2 Extractors

### Step 1: Build Phase 2 Extractor JARs

```bash
cd phase-2/extractor
mvn clean package -DskipTests
# Produces:
#   di/target/rif-extractor-phase2-di-shaded.jar
#   aop/target/rif-extractor-phase2-aop-shaded.jar
#   crossservice/target/rif-extractor-phase2-crossservice-shaded.jar
```

### Step 2: Update Container Image

Bake Phase 2 JARs into the ingestion service container image:

```dockerfile
# In .github/workflows/deploy-ingestion.yml or your container build:
COPY phase-2/extractor/di/target/rif-extractor-phase2-di-shaded.jar /app/rif-extractor-phase2-di.jar
COPY phase-2/extractor/aop/target/rif-extractor-phase2-aop-shaded.jar /app/rif-extractor-phase2-aop.jar
COPY phase-2/extractor/crossservice/target/rif-extractor-phase2-crossservice-shaded.jar /app/rif-extractor-phase2-crossservice.jar
```

### Step 3: Set Environment Variables

In the Container App deployment (or local dev):

```bash
export PHASE2_EXTRACTORS_ENABLED=true
export PHASE2_SOURCE_ROOT=src/main/java
export PHASE2_DI_EXTRACTOR_JAR_PATH=/app/rif-extractor-phase2-di.jar
export PHASE2_AOP_EXTRACTOR_JAR_PATH=/app/rif-extractor-phase2-aop.jar
export PHASE2_CROSSSERVICE_EXTRACTOR_JAR_PATH=/app/rif-extractor-phase2-crossservice.jar
```

See `phase-1/infra/ingestion.containerapp.yaml` for the full deployment manifest.

### Step 4: Test on Stage

Run an end-to-end test against the **stage** database before promoting to production:

```bash
# Stage ingestion service is running, with Phase 2 enabled
curl -X POST http://ingestion-stage:8080/repos/{repo_id}/index \
  -H "Content-Type: application/json" \
  -d '{"sha": ""}'  # Leave empty to resolve from HEAD

# Monitor progress:
curl http://ingestion-stage:8080/repos/{repo_id}/status/{run_id}

# Verify output contains Tier-B/C edges:
psql $STAGE_DATABASE_URL -c "
  SELECT label, COUNT(*) FROM rif.edges()
  WHERE label IN ('INJECTS', 'PRODUCES', 'REGISTERS', 'ADVISES', 'CALLS_SOAP', 'CALLS_REST')
  GROUP BY label;
"
```

### Step 5: Production Deployment

Once stage tests pass:

1. **Update CI/CD**: Enable Phase 2 extractors in the production deployment pipeline
2. **Deploy**: Roll out new container image with Phase 2 JARs + env vars set
3. **Monitor**: Watch ingestion logs for Phase 2 extractor activity:
   ```
   phase2 extractor stdout: ... extractor DI extracted 47 edges
   phase2 extractor stdout: ... extractor AOP extracted 8 edges
   phase2 extractor stdout: ... extractor crossservice extracted 3 edges
   ```

## Validation

After enabling Phase 2 in production:

1. **Provenance gate**: All new edges must have valid `source_ref`
   - Tier-B edges use the extractor's phase-2 format
   - Any UNAVAILABLE nodes fail ingestion (G5 check)

2. **Tier counts**: New runs should report Tier-B and Tier-C edge counts
   ```sql
   SELECT run_id, tier_b_edge_count, tier_c_edge_count
   FROM rif_meta.index_runs
   ORDER BY created_at DESC
   LIMIT 10;
   ```

3. **Graph structure**: Verify new edges are queryable
   ```sql
   SELECT * FROM cypher('rif', '
     MATCH (svc:CLASS)-[inj:INJECTS]->(repo:CLASS)
     WHERE svc.origin = ''first_party''
     RETURN COUNT(inj)
   ') AS (count agtype);
   ```

4. **Embedding service integration** (next step, G2):
   - Phase 2 extractors generate edge embeddings metadata
   - Embedding service (`phase-2/embedding-service`) consumes this and populates `pgvector` columns
   - See G2 for deployment

## Rollback

If issues occur, revert Phase 2 in production:

```bash
# Set environment variable to disable:
export PHASE2_EXTRACTORS_ENABLED=false

# Redeploy (will use Phase 1 extraction only, until re-enabled)
# New runs will populate only Tier-A edges
# Existing Tier-B/C edges in the graph remain live
```

Note: Rolling back does NOT drop Phase 2 edges already in the graph. If you need to roll back the schema:

```bash
# See phase-2/schema/migration_phase2.sql "Rollback" section for details
# Manual steps required to drop AGE labels via ag_catalog.drop_label()
```

## Monitoring

Key metrics to track after enabling Phase 2:

- **Tier-B edges per run**: Should be 10s–100s depending on repo size
- **Tier-C edges per run**: Typically smaller (5–20) unless heavy cross-service calling
- **Extractor runtime**: Each Phase 2 extractor adds 1–2 seconds per run
- **Provenance gap count**: Must remain zero (G5 enforcement)

## Troubleshooting

### Phase 2 extractors not running

Check logs for:
```
"phase2 extractor stdout" OR "phase2 extractor stderr"
```

Ensure:
- JAR files exist at configured paths
- `PHASE2_EXTRACTORS_ENABLED=true`
- Phase 2 source root (`src/main/java`) contains target files

### High resolution failure rate

Phase 2 extractors use SymbolSolver. May fail on:
- Unresolved dependency JARs (not in repo classpath)
- Dynamic proxy classes
- Reflection-based wiring

This is advisory — edges are not dropped, but resolution failures are logged.

### Degenerate extractions

If Phase 2 extractors produce zero edges after schema migration, check:
- Target repo has Spring DI/AOP/cross-service patterns
- SymbolSolver can resolve Spring framework classes
- No unexpected parse errors in extractor stderr

---

**Next step**: G2 (deploy embedding service to ACA + wire ingestion orchestration)
