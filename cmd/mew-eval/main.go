package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"mewcode/internal/evals"
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
	casesDir := flags.String("cases", "", "directory containing evaluation cases")
	outPath := flags.String("out", "", "path for JSONL output")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *mode != "context" && *mode != "teammate" {
		return fmt.Errorf("invalid --mode %q: must be context or teammate", *mode)
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
