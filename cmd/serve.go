package cmd

import (
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"time"

	"github.com/spf13/cobra"

	"github.com/hyperstudyio/hyperstudy-agent-demo/internal/config"
	"github.com/hyperstudyio/hyperstudy-agent-demo/internal/hw"
	"github.com/hyperstudyio/hyperstudy-agent-demo/internal/llama"
	"github.com/hyperstudyio/hyperstudy-agent-demo/internal/models"
)

func lanIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "localhost"
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Launch a HyperStudy-ready llama-server endpoint",
	RunE: func(cmd *cobra.Command, args []string) error {
		modelFlag, _ := cmd.Flags().GetString("model")
		port, _ := cmd.Flags().GetInt("port")
		parallel, _ := cmd.Flags().GetInt("parallel")
		ctx, _ := cmd.Flags().GetInt("ctx")
		regen, _ := cmd.Flags().GetBool("regenerate-key")

		dir := config.DefaultDir()
		cfg, err := config.Load(dir)
		if err != nil {
			return err
		}
		if regen {
			cfg.APIKey = ""
		}
		if cfg.EnsureKey() {
			fmt.Printf("Generated API key (saved to %s/config.json):\n  %s\n\n", dir, cfg.APIKey)
		}
		info, _ := hw.Detect(hw.ExecRunner)
		var ref, name string
		if modelFlag == "" {
			spec := models.Pick(info)
			ref, name = spec.HFRef, spec.Name
		} else {
			spec, isPreset := models.Resolve(modelFlag)
			ref, name = spec.HFRef, spec.Name
			if isPreset && (modelFlag == "gemma4-moe" || modelFlag == "gemma4-4b") {
				fmt.Fprintln(cmd.ErrOrStderr(), "Note: Gemma 4 is a reasoning model — give agents adequate max_tokens (>=256) so it doesn't exhaust its budget before the tool call. Single-turn tool calling works well; a known multi-turn issue (llama.cpp#25072) doesn't affect HyperStudy's single-turn agent decisions. Run `hyperstudy-agent verify` to confirm on your hardware.")
			}
		}
		cfg.Model, cfg.Port = ref, port
		if err := cfg.Save(dir); err != nil {
			return err
		}
		bin, err := llama.Find(exec.LookPath)
		if err != nil {
			return err
		}
		fmt.Printf("Hardware: %s/%s ram=%dGB vram=%dGB spark=%v\nModel:    %s (%s)\n\n", info.OS, info.Arch, info.RAMGB, info.VRAMGB, info.IsSpark(), name, ref)
		proc, err := llama.Start(bin, llama.LaunchOpts{HFRef: ref, APIKey: cfg.APIKey, Port: port, Parallel: parallel, Ctx: ctx})
		if err != nil {
			return err
		}
		// Consume Wait() exactly once, in the background, and hand its
		// result to both the ready-race and the final blocking wait below.
		// Calling proc.Wait() a second time on an already-reaped process
		// returns an error, so nothing else may call it.
		exited := make(chan error, 1)
		go func() { exited <- proc.Wait() }()

		base := fmt.Sprintf("http://localhost:%d", port)
		fmt.Println("Waiting for llama-server (first run downloads the model — may take a while)...")
		if err := llama.WaitReadyOrExit(base, &http.Client{Timeout: 5 * time.Second}, 30*time.Minute, exited); err != nil {
			_ = proc.Process.Kill()
			return err
		}
		fmt.Printf(`
READY
  baseUrl (LAN):   http://%s:%d/v1
  API key:         %s
Next:
  hyperstudy-agent verify                     # prove the endpoint meets the contract
  hyperstudy-agent tunnel                     # get a public URL for HyperStudy
`, lanIP(), port, cfg.APIKey)
		return <-exited
	},
}

func init() {
	serveCmd.Flags().String("model", "", "override the model: a preset (qwen3.6-moe, gemma4-moe, gemma4-4b) or a raw -hf ref (unsloth/Repo-GGUF:Q4_K_M)")
	serveCmd.Flags().Int("port", 8080, "port to serve on")
	serveCmd.Flags().Int("parallel", 8, "concurrent slots (-np)")
	serveCmd.Flags().Int("ctx", 32768, "total context shared across slots (-c)")
	serveCmd.Flags().Bool("regenerate-key", false, "rotate the API key")
	RootCmd.AddCommand(serveCmd)
}
