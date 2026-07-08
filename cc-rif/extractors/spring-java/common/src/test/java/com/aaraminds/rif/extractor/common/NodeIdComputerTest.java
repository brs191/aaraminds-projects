package com.aaraminds.rif.extractor.common;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotEquals;

import org.junit.jupiter.api.Test;

class NodeIdComputerTest {

    @Test
    void computeNodeIdUsesCoreExtractorNulSeparator() {
        String repoId = "repo-a";
        String qualifiedName = "com.example.PaymentService";
        String kind = "CLASS";

        String expected = NodeIdComputer.sha256(repoId + "\u0000" + qualifiedName + "\u0000" + kind);
        String legacySpaceSeparated = NodeIdComputer.sha256(repoId + " " + qualifiedName + " " + kind);

        assertEquals(expected, NodeIdComputer.computeNodeId(repoId, qualifiedName, kind));
        assertNotEquals(legacySpaceSeparated, NodeIdComputer.computeNodeId(repoId, qualifiedName, kind));
    }

    @Test
    void computeEdgeIdUsesCoreExtractorNulSeparator() {
        String from = "a".repeat(64);
        String label = "INJECTS";
        String to = "b".repeat(64);

        String expected = NodeIdComputer.sha256(from + "\u0000" + label + "\u0000" + to);
        String legacySpaceSeparated = NodeIdComputer.sha256(from + " " + label + " " + to);

        assertEquals(expected, NodeIdComputer.computeEdgeId(from, label, to));
        assertNotEquals(legacySpaceSeparated, NodeIdComputer.computeEdgeId(from, label, to));
    }
}
