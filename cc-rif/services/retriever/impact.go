package retriever

import (
	"sort"

	"github.com/aaraminds/rif/graphstore"
)

func impactMetadata(rootID string, br *graphstore.BlastRadiusResult) (map[string]int, map[string]string, map[string]int) {
	depths := map[string]int{rootID: 0}
	tierByNode := map[string]string{}
	outgoing := map[string]int{}
	if br == nil {
		return depths, tierByNode, outgoing
	}

	adj := make(map[string][]graphstore.Edge)
	for _, e := range br.Edges {
		adj[e.FromNodeID] = append(adj[e.FromNodeID], e)
		adj[e.ToNodeID] = append(adj[e.ToNodeID], e)
		outgoing[e.FromNodeID]++
	}

	type frame struct {
		node string
	}
	queue := []frame{{node: rootID}}
	parent := map[string]string{}
	parentEdge := map[string]graphstore.Edge{}
	visited := map[string]struct{}{rootID: {}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, e := range adj[current.node] {
			next := e.ToNodeID
			if next == current.node {
				next = e.FromNodeID
			}
			if _, ok := visited[next]; ok {
				continue
			}
			visited[next] = struct{}{}
			parent[next] = current.node
			parentEdge[next] = e
			depths[next] = depths[current.node] + 1
			queue = append(queue, frame{node: next})
		}
	}

	for nodeID := range depths {
		if nodeID == rootID {
			continue
		}
		tierByNode[nodeID] = classifyTier(nodeID, parent, parentEdge, rootID)
	}

	return depths, tierByNode, outgoing
}

func classifyTier(nodeID string, parent map[string]string, parentEdge map[string]graphstore.Edge, rootID string) string {
	var labels []string
	for cur := nodeID; cur != "" && cur != rootID; cur = parent[cur] {
		if e, ok := parentEdge[cur]; ok {
			labels = append(labels, e.Label)
		} else {
			break
		}
	}
	if len(labels) == 0 {
		return "static"
	}

	sort.Strings(labels)
	for _, label := range labels {
		switch label {
		case "CALLS_SOAP", "CALLS_REST":
			return "cross-service"
		case "ADVISES":
			return "inferred-aop"
		case "INJECTS", "PRODUCES", "REGISTERS":
			return "inferred-di"
		}
	}
	return "static"
}

func impactCaveat(tier string) string {
	switch tier {
	case "static":
		return "Exact AST edges only; reflection and runtime-wired edges may be absent."
	case "inferred-di":
		return "Annotation-based DI only; conditional beans and programmatic registration may be missed."
	case "cross-service":
		return "Static client/stub usage only; runtime endpoint resolution may be incomplete."
	case "inferred-aop":
		return "Static pointcut match only; runtime proxy targets may differ."
	default:
		return "Graph reachability is bounded and may be incomplete."
	}
}
