# urfave-help

[![Tests](https://img.shields.io/github/actions/workflow/status/zigai/urfave-help/test.yml?label=Tests)](https://github.com/zigai/urfave-help/actions/workflows/test.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/zigai/urfave-help.svg)](https://pkg.go.dev/github.com/zigai/urfave-help)
[![Go version](https://img.shields.io/github/go-mod/go-version/zigai/urfave-help)](https://github.com/zigai/urfave-help/blob/master/go.mod)
[![License: MIT](https://img.shields.io/github/license/zigai/urfave-help)](https://github.com/zigai/urfave-help/blob/master/LICENSE)

A drop-in, custom help formatter for urfave/cli/v3.

Partially inspired by Cobra, it gives your CLI clean, readable terminal help:

- **Aligned & wrapped**: Single-column gutters and terminal-aware wrapping with hanging indents.
- **Positional arguments**: First-class argument documentation via `SetArgs`.
- **Clean flags**: Informative placeholders (`<path>`) and defaults without stutter or path leaks.
- **Preserves examples**: Leaves authored multi-line descriptions and code blocks untouched.

---

## Preview

Running the example below outputs:

```text
Manage background tasks and worker deployments

Usage:
  taskctl [command] [flags] <environment> [replicas]

Arguments:
  <environment>           Target environment (staging, production)
  [replicas]              Worker instances (default: 2)

Commands:
  run     Start worker process and execute queued jobs
  status  Display active task status and worker health

Flags:
  --config <path>         Configuration file path
  --timeout <duration>    Job execution duration (default: 30s)
  --format <table|plain>  Output format: table|plain (default: table)
  --filter <string>       Filter jobs by status, tag, priority, queue name, or
                          worker assignment across the cluster
  --dry-run, -n           Simulate task execution without applying changes
  --verbose, -v           Enable debug logging
  --help, -h              Show help
  --version               Print the version

Use "taskctl [command] --help" for more information about a command.
```

---

## Installation

```sh
go get github.com/zigai/urfave-help
```

---

## Example

```go
package main

import (
	"context"
	"os"
	"time"

	"github.com/urfave/cli/v3"

	urfavehelp "github.com/zigai/urfave-help"
)

func main() {
	cmd := &cli.Command{
		Name:      "taskctl",
		Usage:     "Manage background tasks and worker deployments",
		ArgsUsage: "<environment> [replicas]",
		Version:   "1.0.0",
		Commands: []*cli.Command{
			{
				Name:  "run",
				Usage: "Start worker process and execute queued jobs",
			},
			{
				Name:  "status",
				Usage: "Display active task status and worker health",
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config",
				Usage: "Configuration file `path`",
			},
			&cli.DurationFlag{
				Name:  "timeout",
				Value: 30 * time.Second,
				Usage: "Job execution `duration`",
			},
			&cli.StringFlag{
				Name:  "format",
				Value: "table",
				Usage: "Output format: `table|plain`",
			},
			&cli.StringFlag{
				Name:  "filter",
				Usage: "Filter jobs by status, tag, priority, queue name, or worker assignment across the cluster",
			},
			&cli.BoolFlag{
				Name:    "dry-run",
				Aliases: []string{"n"},
				Usage:   "Simulate task execution without applying changes",
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "Enable debug logging",
			},
		},
	}

	urfavehelp.SetArgs(cmd,
		urfavehelp.Arg{Name: "<environment>", Desc: "Target environment (staging, production)"},
		urfavehelp.Arg{Name: "[replicas]", Desc: "Worker instances (default: 2)"},
	)

	urfavehelp.Install(
		urfavehelp.WithWidths(60, 100, 80),
		urfavehelp.WithGutterGap(2),
	)

	_ = cmd.Run(context.Background(), os.Args)
}
```

---

## License

[MIT](LICENSE)
