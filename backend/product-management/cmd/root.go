package cmd

import (
	"context"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "product-management",
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(outboxCmd)
	rootCmd.AddCommand(shardsRootCmd)
}

func Run(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}
