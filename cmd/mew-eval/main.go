package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"mewcode/internal/config"
	"mewcode/internal/evals"
	"mewcode/internal/llm"
	"mewcode/internal/memory"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("mew-eval", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	mode := flags.String("mode", "", "evaluation mode: context or teammate")
	runner := flags.String("runner", "smoke", "evaluation runner: smoke or live")
	casesDir := flags.String("cases", "", "directory containing evaluation cases")
	outPath := flags.String("out", "", "path for JSONL output")
	configPath := flags.String("config", "", "config file path for live evaluation")
	providerName := flags.String("provider", "", "provider name for live evaluation")
	userID := flags.String("user-id", "eval-user", "memory user id for live evaluation")
	appID := flags.String("app-id", "mewcode-eval", "memory app id for live evaluation")
	sessionPrefix := flags.String("session-prefix", "", "session id prefix for live evaluation")
	requireMem0 := flags.Bool("require-mem0", false, "require mem0-platform backend for live evaluation")
	maxCases := flags.Int("max-cases", 0, "maximum number of cases to run")
	timeout := flags.Duration("timeout", 2*time.Minute, "timeout per live case")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *mode != "context" && *mode != "teammate" {
		return fmt.Errorf("invalid --mode %q: must be context or teammate", *mode)
	}
	if *runner != "smoke" && *runner != "live" {
		return fmt.Errorf("invalid --runner %q: must be smoke or live", *runner)
	}
	if *runner == "live" && *mode == "teammate" {
		return fmt.Errorf("live teammate eval is not implemented")
	}
	if *runner == "live" && *configPath == "" {
		return fmt.Errorf("--config is required for live eval")
	}
	if *casesDir == "" {
		return fmt.Errorf("--cases is required")
	}

	var output io.Writer = os.Stdout
	var file *os.File
	if *outPath != "" {
		var err error
		file, err = os.Create(*outPath)
		if err != nil {
			return err
		}
		defer file.Close()
		output = file
	}

	encoder := json.NewEncoder(output)
	if *runner == "live" {
		return runLiveContextEval(liveContextEvalOptions{
			CasesDir:      *casesDir,
			Encoder:       encoder,
			ConfigPath:    *configPath,
			ProviderName:  *providerName,
			UserID:        *userID,
			AppID:         *appID,
			SessionPrefix: *sessionPrefix,
			RequireMem0:   *requireMem0,
			MaxCases:      *maxCases,
			Timeout:       *timeout,
		})
	}

	if *mode == "context" {
		cases, err := evals.LoadContextEvalCases(*casesDir)
		if err != nil {
			return err
		}
		for _, tc := range cases {
			selected := make([]evals.RankedContext, len(tc.RequiredContextIDs))
			for index, id := range tc.RequiredContextIDs {
				selected[index] = evals.RankedContext{ID: id, Rank: index + 1}
			}
			if err := encoder.Encode(evals.ScoreContextEval(tc, selected)); err != nil {
				return err
			}
		}
		return nil
	}

	cases, err := evals.LoadTeammateEvalCases(*casesDir)
	if err != nil {
		return err
	}
	for _, tc := range cases {
		obs := evals.TeammateEvalObservation{
			ToolCalls:       tc.ExpectedToolCalls,
			Messages:        tc.ExpectedMessages,
			TaskTransitions: tc.ExpectedTaskTransitions,
		}
		if err := encoder.Encode(evals.ScoreTeammateEval(tc, obs)); err != nil {
			return err
		}
	}
	return nil
}

type liveContextEvalOptions struct {
	CasesDir      string
	Encoder       *json.Encoder
	ConfigPath    string
	ProviderName  string
	UserID        string
	AppID         string
	SessionPrefix string
	RequireMem0   bool
	MaxCases      int
	Timeout       time.Duration
}

func runLiveContextEval(opts liveContextEvalOptions) error {
	cases, err := evals.LoadContextEvalCases(opts.CasesDir)
	if err != nil {
		return err
	}
	if opts.MaxCases > 0 && opts.MaxCases < len(cases) {
		cases = cases[:opts.MaxCases]
	}
	cfg, err := config.LoadConfig(opts.ConfigPath)
	if err != nil {
		return err
	}
	provider, err := selectProvider(cfg.Providers, opts.ProviderName)
	if err != nil {
		return err
	}
	if !cfg.Memory.LongTerm.Enabled || cfg.Memory.LongTerm.Backend != "mem0-platform" {
		return fmt.Errorf("live context eval requires mem0-platform memory backend")
	}
	store, err := memory.NewMemoryStoreFromConfig(cfg.Memory)
	if err != nil {
		return err
	}
	client, err := llm.NewClient(provider, "")
	if err != nil {
		return err
	}

	runID := opts.SessionPrefix
	if runID == "" {
		runID = "eval-" + time.Now().UTC().Format("20060102150405")
	}
	projectRoot, _ := os.Getwd()
	runner := evals.NewContextLiveRunner(evals.ContextLiveRunnerOptions{
		Store:           store,
		Client:          client,
		Provider:        provider.Name,
		Model:           provider.Model,
		Protocol:        provider.Protocol,
		UserID:          opts.UserID,
		AppID:           opts.AppID,
		AgentID:         "main",
		ProjectID:       projectRoot,
		RunID:           runID,
		ContextWindow:   provider.GetContextWindow(),
		MaxOutputTokens: provider.GetMaxOutputTokens(),
		TokenBudget:     cfg.Memory.LongTerm.Recall.TokenBudget,
		TopK:            cfg.Memory.LongTerm.Recall.TopK,
		Threshold:       cfg.Memory.LongTerm.Recall.Threshold,
		Rerank:          cfg.Memory.LongTerm.Recall.Rerank,
		Timeout:         opts.Timeout,
	})

	for _, tc := range cases {
		result := runner.RunCase(context.Background(), tc)
		if err := opts.Encoder.Encode(result); err != nil {
			return err
		}
	}
	return nil
}

func selectProvider(providers []config.ProviderConfig, name string) (*config.ProviderConfig, error) {
	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers configured")
	}
	if name == "" {
		return &providers[0], nil
	}
	for index := range providers {
		if providers[index].Name == name {
			return &providers[index], nil
		}
	}
	return nil, fmt.Errorf("provider %q not found", name)
}
