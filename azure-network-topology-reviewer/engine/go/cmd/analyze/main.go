// Command analyze runs the deterministic engine on a topology export and prints
// the findings as JSON. The production MCP server (internal/mcp) wraps the same
// analyze.Analyze call as the analyze_risks tool.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/aaraminds/azure-nettopo-engine/internal/analyze"
	"github.com/aaraminds/azure-nettopo-engine/internal/graph"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: analyze <fixture.json>")
		os.Exit(2)
	}
	fx, err := graph.Load(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	out, _ := json.MarshalIndent(analyze.Analyze(fx), "", "  ")
	fmt.Println(string(out))
}
