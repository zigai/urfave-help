package urfavehelp

import (
	"bytes"
	"context"
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
	if !strings.Contains(out, "Flags:\n  --verbose, -v    enable verbose logging\n  --config <path>  config file path\n") {
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

	if !strings.Contains(out, "Usage:\n  deploy [flags] <environment>\n") {
		t.Errorf("unexpected usage line:\n%s", out)
	}
	if !strings.Contains(out, "Arguments:\n  <environment>    Target tier (staging or production)\n") {
		t.Errorf("missing/malformed arguments section:\n%s", out)
	}
	if !strings.Contains(out, "Flags:\n  --target <name>  target cluster name\n  --dry-run, -n    simulate deployment\n") {
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
				Value: "/tmp/local/bin/app",
				Usage: "app binary `path`",
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
	if !strings.Contains(out, "--interval <duration>  reconciliation duration (default: 300ms)") {
		t.Errorf("missing or malformed interval default:\n%s", out)
	}

	// Dynamic path starting with '/' must be suppressed
	if strings.Contains(out, "/tmp/local/bin/app") {
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

	if !strings.Contains(out, "list, ls, l  list all running items") {
		t.Errorf("command aliases not rendered:\n%s", out)
	}
	//nolint:dupword // command name is 'stop' and usage begins with 'stop'
	if !strings.Contains(out, "stop         stop items") {
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

func TestSubcommandHelpAndPersistentDeduplication(t *testing.T) {
	subCmd := &cli.Command{
		Name:  "sub",
		Usage: "subcommand description",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "local-only", Usage: "local switch"},
			&cli.BoolFlag{Name: "help", Aliases: []string{"h"}, Usage: "show help"},
		},
	}

	rootCmd := &cli.Command{
		Name:  "mytool",
		Usage: "My tool description",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "config", Usage: "config path"},
		},
		Commands: []*cli.Command{subCmd},
	}

	// Execute Run to set up parent/root links and persistent flags properly
	var buf bytes.Buffer
	Install()
	rootCmd.Writer = &buf
	_ = rootCmd.Run(context.Background(), []string{"mytool", "sub", "--help"})
	out := buf.String()

	// --help must NOT be lost on subcommand
	if !strings.Contains(out, "--help, -h") {
		t.Errorf("subcommand lost --help flag:\n%s", out)
	}

	// --config should appear under Global Flags, NOT duplicated under Flags
	flagsBlockIndex := strings.Index(out, "Flags:\n")
	globalFlagsIndex := strings.Index(out, "Global Flags:\n")
	if flagsBlockIndex == -1 || globalFlagsIndex == -1 {
		t.Fatalf("expected both Flags: and Global Flags: sections:\n%s", out)
	}

	localFlagsText := out[flagsBlockIndex:globalFlagsIndex]
	if strings.Contains(localFlagsText, "--config") {
		t.Errorf("persistent flag --config duplicated in local Flags:\n%s", out)
	}
	if !strings.Contains(out[globalFlagsIndex:], "--config <string>  config path") {
		t.Errorf("missing --config under Global Flags:\n%s", out)
	}
}

func TestAuthoredUsageText(t *testing.T) {
	cmd := &cli.Command{
		Name:      "appctl",
		Usage:     "Application controller",
		UsageText: "appctl [options]\n   appctl [options] auth\n   appctl [options] deauth",
	}

	var buf bytes.Buffer
	opts := DefaultOptions()
	PrintHelp(&buf, cmd, opts)
	out := buf.String()

	expected := "Usage:\n  appctl [options]\n   appctl [options] auth\n   appctl [options] deauth\n"
	if !strings.Contains(out, expected) {
		t.Errorf("authored UsageText not preserved:\n%s", out)
	}
}

func TestMultiAliasFlags(t *testing.T) {
	cmd := &cli.Command{
		Name:  "app",
		Usage: "app description",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "version", Aliases: []string{"v", "V"}, Usage: "print version"},
			&cli.StringFlag{Name: "output", Aliases: []string{"out", "o"}, Usage: "output file"},
		},
	}

	var buf bytes.Buffer
	opts := DefaultOptions()
	PrintHelp(&buf, cmd, opts)
	out := buf.String()

	if !strings.Contains(out, "--version, -v, -V") {
		t.Errorf("multi-alias short flags truncated:\n%s", out)
	}
	if !strings.Contains(out, "--output, --out, -o <string>") {
		t.Errorf("multi-alias long flags truncated:\n%s", out)
	}
}

func TestExhaustiveGenericsMetavars(t *testing.T) {
	cmd := &cli.Command{
		Name:  "daemon",
		Usage: "daemon description",
		Flags: []cli.Flag{
			&cli.Int64Flag{Name: "timeout", Aliases: []string{"t"}, Usage: "timeout seconds"},
			&cli.Uint64Flag{Name: "gas-limit", Aliases: []string{"l"}, Usage: "gas limit"},
			&cli.Float64SliceFlag{Name: "rates", Usage: "exchange rates"},
		},
	}

	var buf bytes.Buffer
	opts := DefaultOptions()
	PrintHelp(&buf, cmd, opts)
	out := buf.String()

	if !strings.Contains(out, "--timeout, -t <int>") {
		t.Errorf("Int64Flag missing <int> metavar:\n%s", out)
	}
	if !strings.Contains(out, "--gas-limit, -l <uint>") {
		t.Errorf("Uint64Flag missing <uint> metavar:\n%s", out)
	}
	if !strings.Contains(out, "--rates <float...>") {
		t.Errorf("Float64SliceFlag missing <float...> metavar:\n%s", out)
	}
}

func TestCategoriesPreservation(t *testing.T) {
	cmd := &cli.Command{
		Name:  "appctl",
		Usage: "service manager",
		Commands: []*cli.Command{
			{Name: "status", Usage: "show service status"},
			{Name: "migrate", Category: "Database", Usage: "run database migrations"},
			{Name: "listen", Category: "Network", Usage: "listen on network port"},
		},
	}

	var buf bytes.Buffer
	opts := DefaultOptions()
	PrintHelp(&buf, cmd, opts)
	out := buf.String()

	if !strings.Contains(out, "Database:\n    migrate") || !strings.Contains(out, "Network:\n    listen") {
		t.Errorf("command categories not preserved properly:\n%s", out)
	}
}

func TestCompactAuthorsAndCopyright(t *testing.T) {
	cmd := &cli.Command{
		Name:      "tool",
		Usage:     "tool description",
		Copyright: "2026 Example Corp. All rights reserved.",
		Authors: []any{
			"Jane Doe <jane@example.com>",
			"John Smith <john@example.com>",
		},
	}

	var buf bytes.Buffer
	opts := DefaultOptions()
	PrintHelp(&buf, cmd, opts)
	out := buf.String()

	if !strings.Contains(out, "Authors:\n  Jane Doe <jane@example.com>\n  John Smith <john@example.com>") {
		t.Errorf("compact authors missing or malformed:\n%s", out)
	}
	if !strings.Contains(out, "Copyright: 2026 Example Corp. All rights reserved.") {
		t.Errorf("compact copyright missing or malformed:\n%s", out)
	}
}

func TestUsageAndDescriptionCoexistence(t *testing.T) {
	cmd := &cli.Command{
		Name:        "taskctl",
		Usage:       "Task execution manager",
		Description: "Scalable task execution manager for local and remote worker jobs.",
	}

	var buf bytes.Buffer
	opts := DefaultOptions()
	PrintHelp(&buf, cmd, opts)
	out := buf.String()

	expected := "Task execution manager\n\nScalable task execution manager for local and remote worker jobs.\n\nUsage:\n  taskctl"
	if !strings.HasPrefix(out, expected) {
		t.Errorf("Usage and Description coexistence failed:\n%s", out)
	}
}

func TestSafeDefaultsAndDelimiters(t *testing.T) {
	cmd := &cli.Command{
		Name:  "tool",
		Usage: "tool description",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "delimiter", Value: "/", Usage: "delimiter character"},
			&cli.IntFlag{Name: "port", Value: 0, Usage: "port number"},
			&cli.StringSliceFlag{Name: "features", Value: []string{"a", "b"}, Usage: "enabled features"},
		},
	}

	var buf bytes.Buffer
	opts := DefaultOptions()
	PrintHelp(&buf, cmd, opts)
	out := buf.String()

	// "/" delimiter must not be suppressed
	if !strings.Contains(out, "(default: /)") {
		t.Errorf("single character delimiter suppressed:\n%s", out)
	}
	// 0 port must not be suppressed
	if !strings.Contains(out, "(default: 0)") {
		t.Errorf("zero port default suppressed:\n%s", out)
	}
	// Slice features must have clean intact quotes
	if !strings.Contains(out, `(default: "a", "b")`) {
		t.Errorf("slice default quotes corrupted:\n%s", out)
	}
}

func TestFlagCategoriesGrouping(t *testing.T) {
	cmd := &cli.Command{
		Name:  "appctl",
		Usage: "Application controller",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "config", Usage: "config file path"},
			&cli.StringFlag{Name: "host", Category: "Connection Options", Usage: "server hostname"},
			&cli.IntFlag{Name: "port", Category: "Connection Options", Value: 8080, Usage: "server port"},
			&cli.StringFlag{Name: "log-level", Category: "Logging Options", Value: "info", Usage: "log level"},
		},
	}

	var buf bytes.Buffer
	opts := DefaultOptions()
	PrintHelp(&buf, cmd, opts)
	out := buf.String()

	if !strings.Contains(out, "Flags:\n  --config") || !strings.Contains(out, "config file path") {
		t.Errorf("missing uncategorized flags block:\n%s", out)
	}
	if !strings.Contains(out, "Connection Options:\n  --host") || !strings.Contains(out, "--port") {
		t.Errorf("connection options flag category malformed:\n%s", out)
	}
	if !strings.Contains(out, "Logging Options:\n  --log-level") {
		t.Errorf("logging options flag category malformed:\n%s", out)
	}
}

func TestRequiredFlagIndicator(t *testing.T) {
	cmd := &cli.Command{
		Name:  "appctl",
		Usage: "Application controller",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "token", Required: true, Usage: "authentication token"},
			&cli.StringFlag{Name: "optional", Usage: "optional parameter"},
		},
	}

	var buf bytes.Buffer
	opts := DefaultOptions()
	PrintHelp(&buf, cmd, opts)
	out := buf.String()

	if !strings.Contains(out, "--token <string>") || !strings.Contains(out, "authentication token (required)") {
		t.Errorf("required flag missing (required) indicator:\n%s", out)
	}
}

func TestEscapedQuoteDefaultPreservation(t *testing.T) {
	cmd := &cli.Command{
		Name:  "csvtool",
		Usage: "CSV processor",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "quotechar", Value: `"`, Usage: "quote character"},
		},
	}

	var buf bytes.Buffer
	opts := DefaultOptions()
	PrintHelp(&buf, cmd, opts)
	out := buf.String()

	if !strings.Contains(out, `(default: "\"")`) {
		t.Errorf("escaped quote default corrupted:\n%s", out)
	}
}

func TestCustomTakesValueFlagMetavariableFallback(t *testing.T) {
	cmd := &cli.Command{
		Name:  "appctl",
		Usage: "Application controller",
		Flags: []cli.Flag{
			&cli.GenericFlag{
				Name:  "custom",
				Usage: "custom typed flag",
			},
		},
	}

	var buf bytes.Buffer
	opts := DefaultOptions()
	PrintHelp(&buf, cmd, opts)
	out := buf.String()

	if !strings.Contains(out, "--custom <value>") {
		t.Errorf("takes-value flag without explicit type mapping failed to fall back to <value>:\n%s", out)
	}
}

func TestAuthoredCasingPreserved(t *testing.T) {
	cmd := &cli.Command{
		Name:        "toolctl",
		Usage:       "lowercase usage summary",
		Description: "lowercase description explaining REST API and npm packages.",
		Commands: []*cli.Command{
			{Name: "sub", Usage: "lowercase subcommand summary"},
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "json", Usage: "output json format without capitalization"},
		},
	}

	var buf bytes.Buffer
	opts := DefaultOptions()
	PrintHelp(&buf, cmd, opts)
	out := buf.String()

	if !strings.HasPrefix(out, "lowercase usage summary\n\nlowercase description explaining REST API and npm packages.") {
		t.Errorf("authored casing mutated in header:\n%s", out)
	}
	if !strings.Contains(out, "sub  lowercase subcommand summary") {
		t.Errorf("authored casing mutated in command list:\n%s", out)
	}
	if !strings.Contains(out, "--json  output json format without capitalization") {
		t.Errorf("authored casing mutated in flag description:\n%s", out)
	}
}

func TestBoolWithInverseFlagFormatting(t *testing.T) {
	cmd := &cli.Command{
		Name:  "buildtool",
		Usage: "builder tool",
		Flags: []cli.Flag{
			&cli.BoolWithInverseFlag{
				Name:    "version-check",
				Aliases: []string{"vc"},
				Usage:   "verify version consistency",
			},
		},
	}

	var buf bytes.Buffer
	opts := DefaultOptions()
	PrintHelp(&buf, cmd, opts)
	out := buf.String()

	if !strings.Contains(out, "--[no-]version-check, --vc") {
		t.Errorf("BoolWithInverseFlag failed to format with [no-] prefix:\n%s", out)
	}
}
