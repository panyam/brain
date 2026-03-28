package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/panyam/stack-brain/internal/env"
	"github.com/spf13/cobra"
)

// DagOutput is the JSON structure output by the dag command.
type DagOutput struct {
	Tiers []DagTier          `json:"tiers"`
	Edges map[string][]string `json:"edges"`
	// SemanticEdges holds non-module dependency edges (fork-of, shares-api-contract, etc.).
	// Only populated when an environment is active and has external repos with relationships.
	SemanticEdges map[string][]SemanticEdge `json:"semantic_edges,omitempty"`
	// ExternalNodes lists nodes that come from external repos (not controlled by user).
	ExternalNodes []string `json:"external_nodes,omitempty"`
}

// SemanticEdge represents a soft dependency (not a module import).
type SemanticEdge struct {
	Target       string `json:"target"`
	Relationship string `json:"relationship"`
}

// DagTier is a group of components at the same topological level.
type DagTier struct {
	Level      int      `json:"level"`
	Components []string `json:"components"`
}

func newDagCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dag",
		Short: "Print the dependency DAG with topological tiers",
		Long: `Reads all CAPABILITIES.md files, builds the dependency graph,
and outputs topological tiers (leaves first). Use this to determine
the correct update order.

When an environment is active, external repos and semantic dependencies
(fork-of, shares-api-contract, etc.) are included in the graph.`,
		RunE: runDag,
	}

	cmd.Flags().String("downstream-of", "", "Show only components downstream of this component")

	return cmd
}

func runDag(cmd *cobra.Command, args []string) error {
	roots := discoveryRoots()

	components, err := DiscoverComponents(roots...)
	if err != nil {
		return fmt.Errorf("discovering components: %w", err)
	}

	// Build adjacency: edges[A] = [B, C] means A depends on B and C (hard deps)
	edges := make(map[string][]string)
	allNames := make(map[string]bool)
	for _, c := range components {
		name := strings.ToLower(c.Name)
		allNames[name] = true
		for _, dep := range c.StackDeps {
			depName := strings.ToLower(dep)
			edges[name] = append(edges[name], depName)
			allNames[depName] = true
		}
	}

	// Inject external repos and semantic edges from active environment
	semanticEdges := make(map[string][]SemanticEdge)
	var externalNodes []string

	envName, detectErr := env.Detect()
	if detectErr == nil && envName != "" {
		e, loadErr := env.Load(envName)
		if loadErr == nil {
			externals, _ := e.ListExternals()
			for _, ext := range externals {
				extName := strings.ToLower(ext.Name)
				if !allNames[extName] {
					allNames[extName] = true
					externalNodes = append(externalNodes, extName)
				}

				// If this external repo has a semantic relationship, record it.
				// Semantic edges are informational — they appear in output but
				// DON'T affect topological ordering (only hard deps do that).
				if ext.Relationship != "" {
					// Find which internal repos depend on this external
					// For now, add the external as a node; users declare
					// semantic edges via the relationship field on the external repo.
					semanticEdges[extName] = append(semanticEdges[extName], SemanticEdge{
						Target:       extName,
						Relationship: ext.Relationship,
					})
				}
			}
		}
	}

	downstreamOf, _ := cmd.Flags().GetString("downstream-of")

	if downstreamOf != "" {
		target := strings.ToLower(downstreamOf)
		downstream := findDownstream(target, edges)
		downstream[target] = true

		filtered := make(map[string]bool)
		for n := range downstream {
			filtered[n] = true
		}
		for n := range allNames {
			if !filtered[n] {
				delete(allNames, n)
			}
		}
		for n := range edges {
			if !filtered[n] {
				delete(edges, n)
			}
		}
	}

	// Kahn's algorithm for topological sort into tiers
	// inDegree[X] = number of hard dependencies X has (within the known set)
	inDegree := make(map[string]int)
	for n := range allNames {
		inDegree[n] = 0
	}
	for name := range allNames {
		for _, dep := range edges[name] {
			if allNames[dep] {
				inDegree[name]++
			}
		}
	}

	var tiers []DagTier
	remaining := make(map[string]bool)
	for n := range allNames {
		remaining[n] = true
	}

	level := 0
	for len(remaining) > 0 {
		var tier []string
		for n := range remaining {
			if inDegree[n] == 0 {
				tier = append(tier, n)
			}
		}

		if len(tier) == 0 {
			// Cycle detected — dump remaining as final tier
			for n := range remaining {
				tier = append(tier, n)
			}
			tiers = append(tiers, DagTier{Level: level, Components: tier})
			break
		}

		tiers = append(tiers, DagTier{Level: level, Components: tier})

		for _, n := range tier {
			delete(remaining, n)
			for other := range remaining {
				for _, dep := range edges[other] {
					if dep == n {
						inDegree[other]--
					}
				}
			}
		}

		level++
	}

	// Build readable edges map
	readableEdges := make(map[string][]string)
	for name, deps := range edges {
		if len(deps) > 0 {
			readableEdges[name] = deps
		}
	}

	output := DagOutput{
		Tiers: tiers,
		Edges: readableEdges,
	}
	if len(semanticEdges) > 0 {
		output.SemanticEdges = semanticEdges
	}
	if len(externalNodes) > 0 {
		output.ExternalNodes = externalNodes
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

// findDownstream returns all components that transitively depend on target.
func findDownstream(target string, edges map[string][]string) map[string]bool {
	// Build reverse graph: reverseEdges[A] = components that depend on A
	reverse := make(map[string][]string)
	for name, deps := range edges {
		for _, dep := range deps {
			reverse[dep] = append(reverse[dep], name)
		}
	}

	// BFS from target through reverse edges
	visited := make(map[string]bool)
	queue := []string{target}
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		for _, dependent := range reverse[curr] {
			if !visited[dependent] {
				visited[dependent] = true
				queue = append(queue, dependent)
			}
		}
	}

	return visited
}
