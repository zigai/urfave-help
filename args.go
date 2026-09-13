package urfavehelp

import (
	"github.com/urfave/cli/v3"
)

// DefaultArgsMetadataKey stores positional argument documentation in Command.Metadata.
const DefaultArgsMetadataKey = "urfave_help_arguments"

// LegacyArgsMetadataKey is a fallback key for backward compatibility.
const LegacyArgsMetadataKey = "aht_help_arguments"

// Arg documents a positional command argument.
type Arg struct {
	Name string
	Desc string
}

// SetArgs attaches positional argument documentation to a command.
func SetArgs(cmd *cli.Command, args ...Arg) {
	if cmd == nil {
		return
	}
	if cmd.Metadata == nil {
		cmd.Metadata = make(map[string]any)
	}

	cmd.Metadata[DefaultArgsMetadataKey] = args
}

// WithArgs returns a mutator that attaches positional argument documentation.
func WithArgs(args ...Arg) func(*cli.Command) {
	return func(cmd *cli.Command) {
		SetArgs(cmd, args...)
	}
}

// getArgs returns documented positional arguments or falls back to native cmd.Arguments.
func getArgs(cmd *cli.Command) []Arg {
	if cmd == nil {
		return nil
	}

	if cmd.Metadata != nil {
		if v, ok := cmd.Metadata[DefaultArgsMetadataKey]; ok {
			if args, ok := v.([]Arg); ok {
				return args
			}
		}

		if v, ok := cmd.Metadata[LegacyArgsMetadataKey]; ok {
			switch val := v.(type) {
			case []Arg:
				return val
			case []struct{ Name, Desc string }:
				res := make([]Arg, len(val))
				for i, a := range val {
					res[i] = Arg{Name: a.Name, Desc: a.Desc}
				}

				return res
			}
		}
	}

	if len(cmd.Arguments) > 0 {
		args := make([]Arg, 0, len(cmd.Arguments))
		for _, arg := range cmd.Arguments {
			args = append(args, Arg{
				Name: arg.Usage(),
				Desc: "",
			})
		}

		return args
	}

	return nil
}
