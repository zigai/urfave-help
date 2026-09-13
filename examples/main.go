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
