package cmd

import (
	"log/slog"
	"os"

	"github.com/molpako/erasure-coding/store"

	"github.com/spf13/cobra"
)

var loadCmd = &cobra.Command{
	Use: "load",
	Run: func(cmd *cobra.Command, args []string) {
		s, err := store.New(DataBlocks, ParityBlocks, BaseDir)
		if err != nil {
			slog.Error("failed to create store", "error", err)
			os.Exit(1)
		}
		if err := s.Get(args[0], os.Stdout); err != nil {
			slog.Error("failed to load file", "error", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(loadCmd)
}
