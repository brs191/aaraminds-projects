package com.aaraminds.rif.extractor.model;

import java.util.LinkedHashMap;
import java.util.Map;
import java.util.concurrent.atomic.AtomicInteger;

public class RunMetrics {
    public final AtomicInteger filesDiscovered = new AtomicInteger();
    public final AtomicInteger filesParsed = new AtomicInteger();
    public final AtomicInteger filesFailed = new AtomicInteger();
    public final AtomicInteger nodesEmitted = new AtomicInteger();
    public final AtomicInteger edgesEmitted = new AtomicInteger();
    public final AtomicInteger unresolvedTypeCount = new AtomicInteger();
    public final AtomicInteger unresolvedParamTypeCount = new AtomicInteger();
    public final AtomicInteger sameFileResolutionFailureCount = new AtomicInteger();
    public final AtomicInteger resolutionOverflowCount = new AtomicInteger();
    public final AtomicInteger provenanceGapCount = new AtomicInteger();
    public final AtomicInteger unsupportedConstructCount = new AtomicInteger();

    private final long startNanos = System.nanoTime();
    private volatile long elapsedMs;

    public void finish() {
        elapsedMs = (System.nanoTime() - startNanos) / 1_000_000L;
    }

    public void setNodesEmitted(int value) {
        nodesEmitted.set(value);
    }

    public void setEdgesEmitted(int value) {
        edgesEmitted.set(value);
    }

    public Map<String, Object> toMap() {
        LinkedHashMap<String, Object> map = new LinkedHashMap<>();
        map.put("files_discovered", filesDiscovered.get());
        map.put("files_parsed", filesParsed.get());
        map.put("files_failed", filesFailed.get());
        map.put("nodes_emitted", nodesEmitted.get());
        map.put("edges_emitted", edgesEmitted.get());
        map.put("unresolved_type_count", unresolvedTypeCount.get());
        map.put("unresolved_param_type_count", unresolvedParamTypeCount.get());
        map.put("same_file_resolution_failure_count", sameFileResolutionFailureCount.get());
        map.put("resolution_overflow_count", resolutionOverflowCount.get());
        map.put("provenance_gap_count", provenanceGapCount.get());
        map.put("unsupported_construct_count", unsupportedConstructCount.get());
        map.put("elapsed_ms", elapsedMs);
        return map;
    }
}
