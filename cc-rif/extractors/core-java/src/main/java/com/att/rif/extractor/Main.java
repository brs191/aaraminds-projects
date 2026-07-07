package com.att.rif.extractor;

import ch.qos.logback.classic.Level;
import ch.qos.logback.classic.LoggerContext;
import com.att.rif.extractor.model.RunMetrics;
import com.att.rif.extractor.pipeline.DiscoveryStage;
import com.att.rif.extractor.pipeline.EmitStage;
import com.att.rif.extractor.pipeline.ExtractionResult;
import com.att.rif.extractor.pipeline.ParseStage;
import com.att.rif.extractor.pipeline.ResolveStage;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.github.javaparser.ParserConfiguration;
import java.nio.file.Path;
import java.util.List;
import java.util.concurrent.Callable;
import org.slf4j.LoggerFactory;
import picocli.CommandLine;
import picocli.CommandLine.Command;
import picocli.CommandLine.Option;

@Command(name = "rif-extractor", mixinStandardHelpOptions = true, description = "Extracts NDJSON graph data from Java repositories")
public class Main implements Callable<Integer> {

    @Option(names = "--repo-path", required = true)
    private Path repoPath;

    @Option(names = "--repo-id", required = true)
    private String repoId;

    @Option(names = "--sha", required = true)
    private String sha;

    @Option(names = "--output", required = true)
    private Path output;

    @Option(names = "--skip-tests")
    private boolean skipTests;

    @Option(names = "--deps-path")
    private Path depsPath;

    @Option(names = "--verbose")
    private boolean verbose;

    @Option(names = "--files", split = ",",
            description = "Optional repo-relative .java paths for incremental extraction (comma-separated)")
    private List<String> files;

    /**
     * Extraction tier flag.
     * <ul>
     *   <li>{@code A} (default) — Phase 1 Tier-A extraction only (exact, AST-derived).</li>
     *   <li>{@code AB} — Tier-A + Phase 2 Spring DI extractor (INJECTS/PRODUCES/REGISTERS)
     *       + Phase 2 AOP extractor (ADVISES).</li>
     *   <li>{@code ABC} — All of AB plus Phase 2 Cross-Service extractor
     *       (CALLS_SOAP/CALLS_REST).</li>
     * </ul>
     *
     * <p>Phase 2 extractor invocation is implemented as a separate process call to the
     * Phase 2 shaded JARs (see {@code phase-2/extractor/}). Merge and deduplication on
     * {@code node_id} happen at ingest time, not here.
     *
     * <p><b>TODO (Phase 2 integration):</b> Wire actual Phase 2 extractor invocations here
     * once the Phase 2 JARs are co-located. For now {@code AB} and {@code ABC} run Phase 1
     * extraction and log a reminder for the caller to run Phase 2 extractors separately.
     */
    @Option(names = "--tier", defaultValue = "A",
            description = "Extraction tier: A (default, Phase 1 only), AB (+ DI + AOP), ABC (+ cross-service)")
    private String tier;

    @Override
    public Integer call() throws Exception {
        if (verbose) {
            LoggerContext context = (LoggerContext) LoggerFactory.getILoggerFactory();
            context.getLogger("ROOT").setLevel(Level.DEBUG);
        }

        List<Path> incrementalFiles = files == null ? List.of() : files.stream().map(Path::of).toList();
        ExtractorConfig config = new ExtractorConfig(repoPath, repoId, sha, output, skipTests, depsPath, verbose, incrementalFiles);
        RunMetrics metrics = new RunMetrics();
        try {
            DiscoveryStage discoveryStage = new DiscoveryStage(config, metrics);
            List<Path> files = discoveryStage.discover();
            ParseStage parseStage = new ParseStage(config, metrics);
            ParserConfiguration parserConfiguration = parseStage.buildParserConfiguration();
            ResolveStage resolveStage = new ResolveStage(config, metrics, parserConfiguration);
            ExtractionResult result = resolveStage.resolve(files);
            EmitStage emitStage = new EmitStage(config, metrics);
            emitStage.emit(result, output);
            metrics.finish();
            if (!"A".equalsIgnoreCase(tier)) {
                // TODO (Phase 2 integration): invoke Phase 2 extractor JARs and merge output.
                // Until Phase 2 JARs are co-located with the Phase 1 runner, callers using
                // --tier AB or --tier ABC should invoke the Phase 2 extractor modules separately:
                //   rif-extractor-phase2-di   --repo-id ... --sha ... --source-root ... --output ...
                //   rif-extractor-phase2-aop  --repo-id ... --sha ... --source-root ... --output ...
                // and (for ABC):
                //   rif-extractor-phase2-crossservice --repo-id ... --sha ... ...
                // Then merge all NDJSON files deduplicating on node_id before loading to the graph.
                System.err.println("{\"tier_note\":\"Phase 2 extractors (tier=" + tier + ") must be run separately — see phase-2/extractor/\"}");
            }
            System.err.println(new ObjectMapper().writeValueAsString(metrics.toMap()));
            return 0;
        } catch (Exception exception) {
            metrics.finish();
            System.err.println(new ObjectMapper().writeValueAsString(metrics.toMap()));
            throw exception;
        }
    }

    public static void main(String[] args) {
        int exitCode = new CommandLine(new Main()).execute(args);
        System.exit(exitCode);
    }
}
