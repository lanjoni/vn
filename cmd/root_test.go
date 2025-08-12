package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestRootCmd(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd := GetRootCmd()
	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Errorf("Root command should not error on --help: %v", err)
	}

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = old

	output := string(out)

	expectedStrings := []string{
		"VN - Vulnerability Navigator",
		"OWASP Top 10",
		"SQL Injection Testing",
		"sqli",
		"misconfig",
		"help",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected help output to contain '%s', but it didn't.\nOutput: %s", expected, output)
		}
	}
}

func TestRootCmdVersion(t *testing.T) {
	rootCmd := GetRootCmd()
	rootCmd.SetArgs([]string{})

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Errorf("Root command should not error: %v", err)
	}
}

func TestGlobalFlags(t *testing.T) {
	rootCmd := GetRootCmd()
	flags := rootCmd.PersistentFlags()

	verboseFlag := flags.Lookup("verbose")
	if verboseFlag == nil {
		t.Error("Expected verbose flag to be defined")
	}

	outputFlag := flags.Lookup("output")
	if outputFlag == nil {
		t.Error("Expected output flag to be defined")
	}

	configFlag := flags.Lookup("config")
	if configFlag == nil {
		t.Error("Expected config flag to be defined")
	}
}

func TestAvailableCommands(t *testing.T) {
	rootCmd := GetRootCmd()
	commands := rootCmd.Commands()

	commandNames := make([]string, len(commands))
	for i, cmd := range commands {
		commandNames[i] = cmd.Name()
	}

	expectedCommands := []string{"sqli", "xss", "misconfig", "completion", "help"}

	for _, expected := range expectedCommands {
		found := false
		for _, actual := range commandNames {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected command '%s' to be available, but it wasn't. Available: %v", expected, commandNames)
		}
	}
}
