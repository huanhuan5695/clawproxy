package cmd

import (
	"fmt"

	"clawproxy/internal/server"
	"github.com/spf13/cobra"
)

var (
	addr string
)

var rootCmd = &cobra.Command{
	Use:   "clawproxy",
	Short: "WebSocket proxy for openclaw agent command execution",
	RunE: func(cmd *cobra.Command, args []string) error {
		return server.New(addr).Run()
	},
}

func init() {
	rootCmd.Flags().StringVar(&addr, "addr", ":8080", "HTTP listen address")
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		return fmt.Errorf("execute cobra command: %w", err)
	}

	return nil
}
