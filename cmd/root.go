package cmd

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
	"go.dalton.dog/prism/internal"
)

var rootCmd = &cobra.Command{
	Use:   "prism",
	Short: "Prism is a wrapper around go test to make it simple and beautiful",
	Args:  cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		internal.Execute(args)
	},
}

func Execute() {
	if err := fang.Execute(context.Background(), rootCmd, fang.WithoutCompletions()); err != nil {
		os.Exit(1)
	}
}
