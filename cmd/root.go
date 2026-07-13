package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is stamped by goreleaser via -ldflags "-X ...cmd.Version=v1.2.3".
var Version = "dev"

var RootCmd = &cobra.Command{
	Use:     "hyperstudy-agent",
	Short:   "Serve and verify a custom agent endpoint for HyperStudy",
	Version: Version,
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
