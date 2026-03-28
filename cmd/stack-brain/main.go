package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "stack-brain",
		Short: "Stack component discovery and management",
		Long:  "Deterministic operations for the stack brain — lookup, staleness checks, DAG computation, migration collection, and catalog refresh.",
	}

	rootCmd.PersistentFlags().String("brain-dir", "", "Path to brain directory (default: ~/newstack/brain)")
	viper.BindPFlag("brain_dir", rootCmd.PersistentFlags().Lookup("brain-dir"))
	viper.SetDefault("brain_dir", os.ExpandEnv("$HOME/newstack/brain"))
	viper.SetEnvPrefix("STACK_BRAIN")
	viper.AutomaticEnv()

	rootCmd.AddCommand(
		newLookupCmd(),
		newStaleCmd(),
		newDagCmd(),
		newMigrationsCmd(),
		newRefreshCmd(),
		newUpdateCmd(),
		newEnvCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
