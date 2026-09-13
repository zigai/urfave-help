package urfavehelp

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/urfave/cli/v3"
)

func TestBasicRootHelp(t *testing.T) {
	cmd := &cli.Command{
		Name:    "mytool",
		Usage:   "A production service manager",
		Version: "1.0.0",
		Commands: []*cli.Command{
			{Name: "start", Usage: "Start service daemon"},
			{Name: "stop", Usage: "Stop running service"},
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "verbose", Aliases: []string{"v"}, Usage: "enable verbose logging"},
			&cli.StringFlag{Name: "config", Usage: "config file `path`"},
		},
	}

	var buf bytes.Buffer
	opts := DefaultOptions()
	opts.DefaultWidth = 80
	PrintHelp(&buf, cmd, opts)
	out := buf.String()

	if !strings.HasPrefix(out, "A production service manager\n\nUsage:\n  mytool [command] [flags]\n") {
		t.Errorf("unexpected header/usage:\n%s", out)
	}
	if !strings.Contains(out, "Commands:\n  start  Start service daemon\n  stop   Stop running service\n") {
		t.Errorf("unexpected commands block:\n%s", out)
	}
	if !strings.Contains(out, "Flags:\n  --verbose, -v    Enable verbose logging\n  --config <path>  Config file path\n") {
		t.Errorf("unexpected flags block:\n%s", out)
	}
	if !strings.Contains(out, `Use "mytool [command] --help" for more information about a command.`) {
		t.Errorf("missing subcommand footer:\n%s", out)
	}
}

func TestSubcommandHelpAndGutterAlignment(t *testing.T) {
	subCmd := &cli.Command{
		Name:      "deploy",
		Usage:     "Deploy service to cluster",
		ArgsUsage: "<environment>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "target", Usage: "target cluster `name`"},
			&cli.BoolFlag{Name: "dry-run", Aliases: []string{"n"}, Usage: "simulate deployment"},
		},
	}

	SetArgs(subCmd, Arg{
		Name: "<environment>",
		Desc: "Target tier (staging or production)",
	})

	var buf bytes.Buffer
	opts := DefaultOptions()
	opts.DefaultWidth = 80
	PrintHelp(&buf, subCmd, opts)
	out := buf.String()

	if !strings.Contains(out, "Usage:\n  deploy <environment> [flags]\n") {
		t.Errorf("unexpected usage line:\n%s", out)
	}
	if !strings.Contains(out, "Arguments:\n  <environment>    Target tier (staging or production)\n") {
		t.Errorf("missing/malformed arguments section:\n%s", out)
	}
	if !strings.Contains(out, "Flags:\n  --target <name>  Target cluster name\n  --dry-run, -n    Simulate deployment\n") {
		t.Errorf("missing/malformed flags section:\n%s", out)
	}
}

func TestWordWrappingAndHangingIndent(t *testing.T) {
	cmd := &cli.Command{
		Name:  "list",
		Usage: "List items",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "sort",
				Usage: "Sort by: updated, created, harness, presence, activity, cwd, id, multiplexer, tmux, presence-changed, activity-changed",
			},
		},
	}

	var buf bytes.Buffer
	opts := DefaultOptions()
	opts.DefaultWidth = 80
	PrintHelp(&buf, cmd, opts)
	out := buf.String()

	// Verify that the description wrapped across multiple lines
	lines := strings.Split(out, "\n")
	foundFirst := false
	foundSecond := false

	for _, l := range lines {
		if strings.HasPrefix(l, "  --sort <string>  Sort by:") {
			foundFirst = true
		} else if foundFirst && strings.HasPrefix(l, "                   ") && strings.Contains(l, "presence-changed") {
			foundSecond = true
		}
	}

	if !foundFirst || !foundSecond {
		t.Errorf("expected hanging indent wrapping on --sort description:\n%s", out)
	}
}

func TestAdaptiveWideTerminalNoWrapping(t *testing.T) {
	cmd := &cli.Command{
		Name:  "list",
		Usage: "List items",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "sort",
				Usage: "Sort by: updated, created, harness, presence, activity, cwd, id, multiplexer, tmux, presence-changed, activity-changed",
			},
		},
	}

	var buf bytes.Buffer
	opts := DefaultOptions()
	opts.DefaultWidth = 174 // e.g. terminal width 175 - 1
	PrintHelp(&buf, cmd, opts)
	out := buf.String()

	// On a 174-column terminal, the entire sort description must fit on a single line
	expected := "  --sort <string>  Sort by: updated, created, harness, presence, activity, cwd, id, multiplexer, tmux, presence-changed, activity-changed"
	if !strings.Contains(out, expected) {
		t.Errorf("expected single-line sort description on wide terminal, got:\n%s", out)
	}
}

func TestPreservesMultiLineDescriptions(t *testing.T) {
	multiLine := "Run an external worker service.\n\nExamples:\n  mytool run --workers 4\n  mytool run --port 8080\n\nNotes:\n  Requires daemon access."
	cmd := &cli.Command{
		Name:        "worker",
		Usage:       "Worker service",
		Description: multiLine,
	}

	var buf bytes.Buffer
	opts := DefaultOptions()
	opts.DefaultWidth = 80
	PrintHelp(&buf, cmd, opts)
	out := buf.String()

	if !strings.Contains(out, multiLine) {
		t.Errorf("multi-line description was mutated or truncated:\n%s", out)
	}
}

func TestDefaultValueSanitization(t *testing.T) {
	cmd := &cli.Command{
		Name:  "run",
		Usage: "Run tracker",
		Flags: []cli.Flag{
			&cli.DurationFlag{
				Name:  "interval",
				Value: 300 * time.Millisecond,
				Usage: "reconciliation `duration`",
			},
			&cli.StringFlag{
				Name:  "binary",
				Value: "/tmp/local/bin/aht",
				Usage: "aht binary `path`",
			},
			&cli.BoolFlag{
				Name:  "quiet",
				Usage: "suppress output",
			},
		},
	}

	var buf bytes.Buffer
	opts := DefaultOptions()
	PrintHelp(&buf, cmd, opts)
	out := buf.String()

	// 300ms static default must appear
	if !strings.Contains(out, "--interval <duration>  Reconciliation duration (default: 300ms)") {
		t.Errorf("missing or malformed interval default:\n%s", out)
	}

	// Dynamic path starting with '/' must be suppressed
	if strings.Contains(out, "/tmp/local/bin/aht") {
		t.Errorf("leaked dynamic binary path default into help:\n%s", out)
	}

	// Boolean switch must not display (default: false)
	if strings.Contains(out, "(default: false)") {
		t.Errorf("leaked boolean default into help:\n%s", out)
	}
}

func TestSubcommandAliasesWidening(t *testing.T) {
	cmd := &cli.Command{
		Name:  "app",
		Usage: "App management",
		Commands: []*cli.Command{
			{Name: "list", Aliases: []string{"ls", "l"}, Usage: "list all running items"},
			{Name: "stop", Usage: "stop items"},
		},
	}

	var buf bytes.Buffer
	opts := DefaultOptions()
	PrintHelp(&buf, cmd, opts)
	out := buf.String()

	if !strings.Contains(out, "list, ls, l  List all running items") {
		t.Errorf("command aliases not rendered:\n%s", out)
	}
	if !strings.Contains(out, "stop         Stop items") {
		t.Errorf("stop command not aligned to widened gutter:\n%s", out)
	}
}

func TestNativeArgumentsFallback(t *testing.T) {
	cmd := &cli.Command{
		Name:  "inspect",
		Usage: "Inspect target",
		Arguments: []cli.Argument{
			&cli.StringArg{Name: "target", UsageText: "<target>"},
		},
	}

	var buf bytes.Buffer
	opts := DefaultOptions()
	PrintHelp(&buf, cmd, opts)
	out := buf.String()

	if !strings.Contains(out, "Arguments:\n  <target>\n") {
		t.Errorf("native argument fallback failed:\n%s", out)
	}
}
