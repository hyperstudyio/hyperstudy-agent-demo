package cmd

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/hyperstudyio/hyperstudy-agent-demo/internal/config"
	"github.com/hyperstudyio/hyperstudy-agent-demo/internal/verify"
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Check an endpoint against the HyperStudy custom-agent contract",
	RunE: func(cmd *cobra.Command, args []string) error {
		baseURL, _ := cmd.Flags().GetString("base-url")
		apiKey, _ := cmd.Flags().GetString("api-key")
		n, _ := cmd.Flags().GetInt("concurrency")
		cfg, err := config.Load(config.DefaultDir())
		if err != nil {
			return err
		}
		if apiKey == "" {
			apiKey = cfg.APIKey
		}
		if baseURL == "" {
			port := cfg.Port
			if port == 0 {
				port = 8080
			}
			baseURL = fmt.Sprintf("http://localhost:%d/v1", port)
		}
		if apiKey == "" {
			return fmt.Errorf("no API key: pass --api-key or run `hyperstudy-agent serve` first")
		}
		fmt.Printf("Verifying %s\n\n", baseURL)
		client := &http.Client{Timeout: 310 * time.Second}
		failed := false
		for _, r := range verify.Run(baseURL, apiKey, n, client) {
			status := "PASS"
			if !r.Pass {
				status, failed = "FAIL", true
			}
			fmt.Printf("  [%s] %-28s %s\n", status, r.Name, r.Detail)
		}
		if failed {
			os.Exit(1)
		}
		fmt.Println("\nAll checks passed — paste the baseUrl and key into HyperStudy Settings → API Keys → Custom Agent Endpoint.")
		return nil
	},
}

func init() {
	verifyCmd.Flags().String("base-url", "", "endpoint base URL including /v1 (default: the served config)")
	verifyCmd.Flags().String("api-key", "", "bearer key (default: the saved key)")
	verifyCmd.Flags().Int("concurrency", 8, "parallel requests for the load smoke test")
	RootCmd.AddCommand(verifyCmd)
}
