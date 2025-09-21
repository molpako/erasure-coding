package cmd

import (
	"log/slog"
	"os"

	"github.com/molpako/erasure-coding/store"

	"github.com/spf13/cobra"
)

var saveCmd = &cobra.Command{
	Use: "save",
	Run: func(cmd *cobra.Command, args []string) {
		f, err := os.Open(args[0])
		if err != nil {
			slog.Error("failed to get filename", "error", err)
			os.Exit(1)
		}
		defer f.Close()

		s, err := store.New(DataBlocks, ParityBlocks, BaseDir)
		if err != nil {
			slog.Error("failed to create store", "error", err)
			os.Exit(1)
		}

		if err = s.Save(f.Name(), f); err != nil {
			slog.Error("failed to save file", "error", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(saveCmd)
}
