package com.aaraminds.rif.extractor.di;

import com.aaraminds.rif.extractor.common.EmitHelper;
import java.nio.file.Path;
import java.util.Map;
import java.util.concurrent.Callable;
import picocli.CommandLine;
import picocli.CommandLine.Command;
import picocli.CommandLine.Option;

/**
 * CLI entry point for the Phase 2 Spring DI extractor.
 *
 * <pre>
 * Usage:
 *   java -jar rif-extractor-phase2-di-shaded.jar \
 *     --repo-id myrepo \
 *     --sha &lt;40-char SHA&gt; \
 *     --source-root /path/to/src/main/java \
 *     --output /path/to/output.ndjson
 * </pre>
 *
 * Emits Tier-B edges: INJECTS, PRODUCES, REGISTERS.
 */
@Command(
        name = "rif-di-extractor",
        mixinStandardHelpOptions = true,
        description = "Phase 2 Spring DI extractor — emits INJECTS, PRODUCES, REGISTERS edges (Tier-B)"
)
public class DiMain implements Callable<Integer> {

    @Option(names = "--repo-id", required = true, description = "Stable repository identifier")
    private String repoId;

    @Option(names = "--sha", required = true, description = "40-character Git commit SHA")
    private String sha;

    @Option(names = "--source-root", required = true, description = "Root of Java source tree (e.g. src/main/java)")
    private Path sourceRoot;

    @Option(names = "--output", required = true, description = "Output NDJSON file path")
    private Path output;

    @Override
    public Integer call() throws Exception {
        SpringDiExtractor extractor = new SpringDiExtractor(repoId, sha, sourceRoot);
        SpringDiExtractor.ExtractionResult result = extractor.extract();
        EmitHelper.emit(result.nodes(), result.edges(), output);
        System.err.printf("{\"nodes\":%d,\"edges\":%d}%n",
                result.nodes().size(), result.edges().size());
        return 0;
    }

    public static void main(String[] args) {
        System.exit(new CommandLine(new DiMain()).execute(args));
    }
}
