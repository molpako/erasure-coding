package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	DataBlocks, ParityBlocks = 4, 2

	BaseDir string
)

var rootCmd = &cobra.Command{
	Use: "erasure-coding",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	p, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	BaseDir = filepath.Join(p, ".workdir", "mnt")
}
