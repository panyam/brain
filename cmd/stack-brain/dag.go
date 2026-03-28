package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

type DagOutput struct {
	Tiers []DagTier          `json:"tiers"`
	Edges map[string][]string `json:"edges"`
}

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
the correct update order.`,
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

	// Build adjacency: edges[A] = [B, C] means A depends on B and C
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

	downstreamOf, _ := cmd.Flags().GetString("downstream-of")

	if downstreamOf != "" {
		// Filter to only downstream components
		target := strings.ToLower(downstreamOf)
		downstream := findDownstream(target, edges, allNames)
		downstream[target] = true

		// Remove non-downstream
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
	inDegree := make(map[string]int)
	for n := range allNames {
		inDegree[n] = 0
	}
	// Reverse edges for "depended-on-by" (we need dependents, not dependencies)
	// Actually for tiers, we want: tier 0 = nodes with no dependencies (leaves)
	for name := range allNames {
		for _, dep := range edges[name] {
			if allNames[dep] {
				inDegree[name]++ // this is wrong direction, let me fix
			}
		}
	}

	// Recompute: inDegree[X] = number of dependencies X has (within the known set)
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
		// Find nodes with inDegree 0 among remaining
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

		// Remove this tier's nodes and update inDegrees
		for _, n := range tier {
			delete(remaining, n)
			// Reduce inDegree for anything that depended on n
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

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

// findDownstream returns all components that transitively depend on target.
func findDownstream(target string, edges map[string][]string, allNames map[string]bool) map[string]bool {
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
