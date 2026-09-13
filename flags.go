package urfavehelp

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"
)

type flagItem struct {
	label string
	desc  string
}

// unquoteUsage extracts backtick-enclosed placeholder text from usage strings.
func unquoteUsage(usage string) (string, string) {
	for i := range len(usage) {
		if usage[i] == '`' {
			for j := i + 1; j < len(usage); j++ {
				if usage[j] == '`' {
					name := usage[i+1 : j]
					clean := usage[:i] + name + usage[j+1:]
					return name, clean
				}
			}

			break
		}
	}
	return "", usage
}

// flagMetavar returns the placeholder for a flag (<value>), or empty for boolean switches.
func flagMetavar(fl cli.Flag, rawUsage string, customResolver func(cli.Flag) string) string {
	if customResolver != nil {
		if custom := customResolver(fl); custom != "" {
			if !strings.HasPrefix(custom, "<") {
				return "<" + custom + ">"
			}
			return custom
		}
	}

	placeholder, _ := unquoteUsage(rawUsage)
	if placeholder != "" {
		return "<" + placeholder + ">"
	}

	switch fl.(type) {
	case *cli.StringFlag:
		return "<string>"
	case *cli.DurationFlag:
		return "<duration>"
	case *cli.IntFlag:
		return "<int>"
	case *cli.UintFlag:
		return "<uint>"
	case *cli.FloatFlag:
		return "<float>"
	case *cli.StringSliceFlag:
		return "<string...>"
	default:
		return ""
	}
}

// formatFlagItem formats a flag into its label and description.
func formatFlagItem(fl cli.Flag, pathPrefixFilter string, customMetavar func(cli.Flag) string) flagItem {
	names := fl.Names()
	var longName, shortName string
	for _, name := range names {
		if len(name) == 1 {
			if shortName == "" {
				shortName = "-" + name
			}
		} else {
			if longName == "" {
				longName = "--" + name
			}
		}
	}

	rawUsage := ""
	if uf, ok := fl.(cli.DocGenerationFlag); ok {
		rawUsage = uf.GetUsage()
	}

	metavar := flagMetavar(fl, rawUsage, customMetavar)

	var labelBuilder strings.Builder
	if longName != "" {
		labelBuilder.WriteString(longName)
		if shortName != "" {
			labelBuilder.WriteString(", " + shortName)
		}
		if metavar != "" {
			labelBuilder.WriteString(" " + metavar)
		}
	} else if shortName != "" {
		labelBuilder.WriteString(shortName)
		if metavar != "" {
			labelBuilder.WriteString(" " + metavar)
		}
	}

	defaultStr, envStr := flagDefaultAndEnv(fl, pathPrefixFilter)
	_, unquoted := unquoteUsage(rawUsage)
	desc := capitalizeFirst(unquoted) + defaultStr + envStr

	return flagItem{
		label: labelBuilder.String(),
		desc:  desc,
	}
}

// flagDefaultAndEnv extracts default value and environment variable hints.
func flagDefaultAndEnv(fl cli.Flag, pathPrefixFilter string) (string, string) {
	df, ok := fl.(cli.DocGenerationFlag)
	if !ok {
		return "", ""
	}

	var defaultStr, envStr string
	if rf, ok := fl.(cli.RequiredFlag); !ok || !rf.IsRequired() {
		defaultStr = flagDefaultText(df, pathPrefixFilter)
	}

	if envVars := df.GetEnvVars(); len(envVars) > 0 {
		envStr = fmt.Sprintf(" [$%s]", strings.Join(envVars, ", $"))
	}

	return defaultStr, envStr
}

// flagDefaultText formats a non-zero default while suppressing paths matching pathPrefixFilter.
func flagDefaultText(df cli.DocGenerationFlag, pathPrefixFilter string) string {
	if !df.IsDefaultVisible() {
		return ""
	}
	if s := df.GetDefaultText(); s != "" {
		return fmt.Sprintf(" (default: %s)", s)
	}
	if !df.TakesValue() {
		return ""
	}

	val := strings.Trim(df.GetValue(), `"`)
	if val == "" || val == "0" || val == "false" || val == "0s" {
		return ""
	}

	if pathPrefixFilter != "" && strings.HasPrefix(val, pathPrefixFilter) {
		return ""
	}

	return fmt.Sprintf(" (default: %s)", val)
}
