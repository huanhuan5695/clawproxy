package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"clawproxy/internal/auth"
	"clawproxy/internal/server"
	"github.com/spf13/cobra"
)

var (
	addr      string
	jwtSecret string
)

var rootCmd = &cobra.Command{
	Use:   "clawproxy",
	Short: "WebSocket proxy for openclaw agent command execution",
	RunE: func(cmd *cobra.Command, args []string) error {
		return server.New(addr, jwtSecret).Run()
	},
}

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Generate a JWT token for /ws authentication",
	RunE: func(cmd *cobra.Command, args []string) error {
		deviceID, err := cmd.Flags().GetString("device-id")
		if err != nil {
			return fmt.Errorf("get device-id flag: %w", err)
		}

		expiresInRaw, err := cmd.Flags().GetString("expires-in")
		if err != nil {
			return fmt.Errorf("get expires-in flag: %w", err)
		}

		expiresIn, err := parseExpiresInDays(expiresInRaw)
		if err != nil {
			return err
		}

		tokenString, err := auth.GenerateToken([]byte(jwtSecret), deviceID, expiresIn)
		if err != nil {
			return fmt.Errorf("generate jwt token: %w", err)
		}

		cmd.Println(tokenString)
		return nil
	},
}

func parseExpiresInDays(raw string) (time.Duration, error) {
	if raw == "" {
		return 0, nil
	}

	if !strings.HasSuffix(raw, "d") {
		return 0, fmt.Errorf("invalid expires-in format: %q, expected format like 1d", raw)
	}

	daysPart := strings.TrimSuffix(raw, "d")
	days, err := strconv.Atoi(daysPart)
	if err != nil || days <= 0 {
		return 0, fmt.Errorf("invalid expires-in value: %q, expected positive integer days like 1d", raw)
	}

	return time.Duration(days) * 24 * time.Hour, nil
}

func init() {
	rootCmd.Flags().StringVar(&addr, "addr", ":8080", "HTTP listen address")
	rootCmd.PersistentFlags().StringVar(&jwtSecret, "jwt-secret", "clawproxy-dev-secret", "JWT shared secret for token verification and generation")

	tokenCmd.Flags().String("device-id", "", "device/session id used as JWT sub claim")
	tokenCmd.Flags().String("expires-in", "", "token expiration in days, e.g. 1d; empty means never expires")
	_ = tokenCmd.MarkFlagRequired("device-id")
	rootCmd.AddCommand(tokenCmd)
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		return fmt.Errorf("execute cobra command: %w", err)
	}

	return nil
}
