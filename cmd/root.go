package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"

	"go.dalton.dog/prism/internal"
)

var (
	Version       = "1.3"
	configLoadErr error
)

var rootCmd = &cobra.Command{
	Use:   "prism",
	Short: "Prism is a wrapper around go test to make it simple and beautiful",
	Long: internal.Header() + "\n\n" + `Prism is a wrapper around Go's built in test command that aims to make it beautiful and organized.

Issues? Requests? Feedback? Let me know! -- github.com/DaltonSW/prism`,
	Args: cobra.ArbitraryArgs,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if internal.GlobalConfig.NoColor || (os.Getenv("NO_COLOR") != "" && !internal.GlobalConfig.ShowColor) {
			internal.UnsetColors()
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		if internal.GlobalConfig.Watch {
			internal.Watch(args)
			return
		}
		internal.Execute(args)
	},
}

func Execute() {
	if configLoadErr != nil {
		fmt.Fprintf(os.Stderr, "Unable to load persisted config: %v\n", configLoadErr)
	}

	if err := fang.Execute(context.Background(), rootCmd, fang.WithoutCompletions(), fang.WithVersion(Version)); err != nil {
		os.Exit(1)
	}
}

func init() {
	var cfg internal.Config
	cfg, configLoadErr = internal.LoadConfig()
	internal.GlobalConfig = cfg

	rootCmd.PersistentFlags().BoolVarP(&internal.GlobalConfig.Verbose, "verbose", "v", internal.GlobalConfig.Verbose, "Include test sub-output")
	rootCmd.PersistentFlags().BoolVarP(&internal.GlobalConfig.OnlyFails, "only-fails", "f", internal.GlobalConfig.OnlyFails, "Only run failing tests")
	rootCmd.PersistentFlags().BoolVar(&internal.GlobalConfig.NoBar, "no-bar", internal.GlobalConfig.NoBar, "Hide the summary bar at the end of test output")
	rootCmd.PersistentFlags().BoolVarP(&internal.GlobalConfig.Watch, "watch", "w", internal.GlobalConfig.Watch, "Watch for file changes and rerun tests")

	rootCmd.PersistentFlags().BoolVar(&internal.GlobalConfig.NoColor, "no-color", internal.GlobalConfig.NoColor, "Disable color output entirely")
	rootCmd.PersistentFlags().BoolVar(&internal.GlobalConfig.ShowColor, "color", internal.GlobalConfig.ShowColor, "Force color output, overridding NO_COLOR environment variable")
	rootCmd.MarkFlagsMutuallyExclusive("no-color", "color")
}
