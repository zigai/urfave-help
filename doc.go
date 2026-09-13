// Package urfavehelp provides a clean, terminal-aware help formatter for github.com/urfave/cli/v3.
//
// Features:
//   - Aligned columns: arguments, commands, and flags share a single vertical gutter.
//   - Smart word wrapping: reflows descriptions at terminal width with hanging indents.
//   - Clean placeholders: enforces <value> on value flags without repeating flag names.
//   - Positional arguments: document arguments with [SetArgs] or native cli.Command.Arguments.
//   - Sanitized defaults: hides dynamic machine-local paths like [os.Executable].
//   - Subcommand aliases: displays command aliases (e.g. list, ls) with aligned spacing.
//
// # Quick Start
//
// Enable urfavehelp with a single line:
//
//	package main
//
//	import (
//	    "context"
//	    "os"
//
//	    "github.com/urfave/cli/v3"
//	    urfavehelp "github.com/zigai/urfave-help"
//	)
//
//	func main() {
//	    cmd := &cli.Command{
//	        Name:  "mytool",
//	        Usage: "A production command-line interface",
//	    }
//
//	    urfavehelp.Install()
//	    _ = cmd.Run(context.Background(), os.Args)
//	}
package urfavehelp
