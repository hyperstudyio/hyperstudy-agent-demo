package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"

	"github.com/spf13/cobra"

	"github.com/hyperstudyio/hyperstudy-agent/internal/config"
)

var tunnelURLRe = regexp.MustCompile(`https://[a-z0-9-]+\.trycloudflare\.com`)

var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "Expose the local endpoint via a cloudflared quick tunnel",
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		if port == 0 {
			cfg, _ := config.Load(config.DefaultDir())
			port = cfg.Port
			if port == 0 {
				port = 8080
			}
		}
		bin, err := exec.LookPath("cloudflared")
		if err != nil {
			return fmt.Errorf(`cloudflared not found on PATH.
Install it:
  macOS: brew install cloudflared
  Linux: https://github.com/cloudflare/cloudflared/releases
(For a stable hostname across restarts, see Tailscale Funnel in the README.)`)
		}
		proc := exec.Command(bin, "tunnel", "--url", fmt.Sprintf("http://localhost:%d", port))
		stderr, err := proc.StderrPipe()
		if err != nil {
			return err
		}
		if err := proc.Start(); err != nil {
			return err
		}
		scanner := bufio.NewScanner(stderr)
		printed := false
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Fprintln(os.Stderr, line)
			if printed {
				continue
			}
			if url := tunnelURLRe.FindString(line); url != "" {
				printed = true
				fmt.Printf(`
PUBLIC ENDPOINT
  baseUrl: %s/v1
Next:
  hyperstudy-agent verify --base-url %s/v1
Then paste the baseUrl + your API key into HyperStudy Settings → API Keys.
(Quick tunnels get a NEW hostname each run — re-save the credential if you restart.)
`, url, url)
			}
		}
		return proc.Wait()
	},
}

func init() {
	tunnelCmd.Flags().Int("port", 0, "local port to expose (default: the served config)")
	RootCmd.AddCommand(tunnelCmd)
}
