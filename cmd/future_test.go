package cmd

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestRunFutureValuePrintsAllHorizons(t *testing.T) {
	output := captureFutureValueOutput(t, "100.00")

	for _, want := range []string{
		"Future Value (5% Annual Interest)",
		"Starting value: 100.00",
		"5 years: 127.63",
		"10 years: 162.89",
		"15 years: 207.89",
		"20 years: 265.33",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("future output does not contain %q:\n%s", want, output)
		}
	}
}

func TestRunFutureValueRejectsNonPositiveValues(t *testing.T) {
	for _, input := range []string{"0", "-1.00"} {
		t.Run(input, func(t *testing.T) {
			if err := runFutureValue(input); err == nil {
				t.Errorf("runFutureValue(%q) returned nil error", input)
			}
		})
	}
}

func captureFutureValueOutput(t *testing.T, input string) string {
	t.Helper()

	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	err = runFutureValue(input)
	w.Close()
	os.Stdout = original
	if err != nil {
		t.Fatalf("runFutureValue(%q): %v", input, err)
	}

	output, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read captured output: %v", err)
	}
	return string(output)
}
