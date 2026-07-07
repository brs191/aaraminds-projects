package com.att.rif.extractor.crossservice;

import com.att.rif.extractor.common.EmitHelper;
import java.nio.file.Path;
import java.util.concurrent.Callable;
import picocli.CommandLine;
import picocli.CommandLine.Command;
import picocli.CommandLine.Option;

/**
 * CLI entry point for the Phase 2 Cross-Service extractor.
 *
 * <pre>
 * Usage:
 *   java -jar rif-extractor-phase2-crossservice-shaded.jar \
 *     --repo-id myrepo \
 *     --sha &lt;40-char SHA&gt; \
 *     --source-root /path/to/src/main/java \
 *     --output /path/to/output.ndjson
 * </pre>
 *
 * Emits Tier-C edges: CALLS_SOAP, CALLS_REST.
 */
@Command(
        name = "rif-crossservice-extractor",
        mixinStandardHelpOptions = true,
        description = "Phase 2 Cross-Service extractor — emits CALLS_SOAP and CALLS_REST edges (Tier-C)"
)
public class CrossServiceMain implements Callable<Integer> {

    @Option(names = "--repo-id", required = true, description = "Stable repository identifier")
    private String repoId;

    @Option(names = "--sha", required = true, description = "40-character Git commit SHA")
    private String sha;

    @Option(names = "--source-root", required = true, description = "Root of Java source tree")
    private Path sourceRoot;

    @Option(names = "--output", required = true, description = "Output NDJSON file path")
    private Path output;

    @Override
    public Integer call() throws Exception {
        CrossServiceExtractor extractor = new CrossServiceExtractor(repoId, sha, sourceRoot);
        CrossServiceExtractor.ExtractionResult result = extractor.extract();
        EmitHelper.emit(result.nodes(), result.edges(), output);
        System.err.printf("{\"nodes\":%d,\"edges\":%d}%n",
                result.nodes().size(), result.edges().size());
        return 0;
    }

    public static void main(String[] args) {
        System.exit(new CommandLine(new CrossServiceMain()).execute(args));
    }
}
