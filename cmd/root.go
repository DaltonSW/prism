package cmd

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "prism",
	Short: "Prism is a wrapper around go test to make it simple and beautiful",
	Run: func(cmd *cobra.Command, args []string) {
		// Do some stuff
	},
}

func Execute() {
	if err := fang.Execute(context.Background(), rootCmd, fang.WithoutCompletions()); err != nil {
		os.Exit(1)
	}
}
