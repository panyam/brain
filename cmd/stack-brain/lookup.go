package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type LookupResult struct {
	Component    string       `json:"component"`
	Module       string       `json:"module"`
	Location     string       `json:"location"`
	Version      string       `json:"version"`
	Status       string       `json:"status"`
	Score        int          `json:"score"`
	Matches      []Capability `json:"matches"`
	StackDeps    []string     `json:"stack_deps,omitempty"`
}

func newLookupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lookup [phrases...]",
		Short: "Search components by keywords or phrases",
		Long: `Search all stack components by matching keywords and phrases against
capability tags, descriptions, module names, and sub-component metadata.

Multiple arguments are treated as separate search phrases. Each phrase is
matched independently — a component's score is the number of phrases it matches.

Examples:
  stack-brain lookup caching
  stack-brain lookup "federated auth" signups wasm
  stack-brain lookup "typescript client"`,
		Args: cobra.MinimumNArgs(1),
		RunE: runLookup,
	}

	cmd.Flags().Int("top", 0, "Limit to top N results (0 = all matches)")

	return cmd
}

func runLookup(cmd *cobra.Command, args []string) error {
	brainDir := viper.GetString("brain_dir")
	stackRoot := expandHome(brainDir + "/..")

	components, err := DiscoverComponents(stackRoot)
	if err != nil {
		return fmt.Errorf("discovering components: %w", err)
	}

	type scored struct {
		comp  *Component
		score int
	}

	var results []scored
	for _, comp := range components {
		s := comp.Match(args)
		if s > 0 {
			results = append(results, scored{comp: comp, score: s})
		}
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	top, _ := cmd.Flags().GetInt("top")
	if top > 0 && len(results) > top {
		results = results[:top]
	}

	// Build output
	var output []LookupResult
	for _, r := range results {
		lr := LookupResult{
			Component: r.comp.Name,
			Module:    r.comp.Module,
			Location:  r.comp.Location,
			Version:   r.comp.Version,
			Status:    r.comp.Status,
			Score:     r.score,
			StackDeps: r.comp.StackDeps,
		}
		// Include only matching capabilities
		for _, cap := range r.comp.Capabilities {
			if capMatchesAny(cap, args) {
				lr.Matches = append(lr.Matches, cap)
			}
		}
		output = append(output, lr)
	}

	if len(output) == 0 {
		fmt.Fprintln(os.Stderr, "no matches found")
		return nil
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

// capMatchesAny checks if a capability matches any of the query phrases.
func capMatchesAny(cap Capability, phrases []string) bool {
	text := lowercase(cap.Tag + " " + cap.Description + " " + cap.SubModule + " " + cap.SubLocation)
	for _, p := range phrases {
		if containsLower(text, p) {
			return true
		}
	}
	return false
}
