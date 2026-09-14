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
//
//nolint:cyclop // comprehensive type switch across all urfave/cli flag variants
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
	case *cli.IntFlag, *cli.Int8Flag, *cli.Int16Flag, *cli.Int32Flag, *cli.Int64Flag:
		return "<int>"
	case *cli.UintFlag, *cli.Uint8Flag, *cli.Uint16Flag, *cli.Uint32Flag, *cli.Uint64Flag:
		return "<uint>"
	case *cli.FloatFlag, *cli.Float32Flag:
		return "<float>"
	case *cli.StringSliceFlag:
		return "<string...>"
	case *cli.IntSliceFlag, *cli.Int8SliceFlag, *cli.Int16SliceFlag, *cli.Int32SliceFlag, *cli.Int64SliceFlag:
		return "<int...>"
	case *cli.UintSliceFlag, *cli.Uint16SliceFlag, *cli.Uint32SliceFlag, *cli.Uint64SliceFlag:
		return "<uint...>"
	case *cli.FloatSliceFlag, *cli.Float32SliceFlag:
		return "<float...>"
	case *cli.TimestampFlag:
		return "<time>"
	}

	if df, ok := fl.(cli.DocGenerationFlag); ok && df.TakesValue() {
		return "<value>"
	}

	return ""
}

// formatFlagItem formats a flag into its label and description.
func formatFlagItem(fl cli.Flag, pathPrefixFilter string, customMetavar func(cli.Flag) string) flagItem {
	names := fl.Names()
	var longNames, shortNames []string
	isInverse := false
	invPrefix := ""
	if bif, ok := fl.(*cli.BoolWithInverseFlag); ok {
		isInverse = true
		invPrefix = bif.InversePrefix
		if invPrefix == "" {
			invPrefix = cli.DefaultInverseBoolPrefix
		}
	}

	for i, name := range names {
		if len(name) == 1 {
			shortNames = append(shortNames, "-"+name)
		} else {
			if isInverse && i == 0 {
				longNames = append(longNames, fmt.Sprintf("--[%s]%s", invPrefix, name))
			} else {
				longNames = append(longNames, "--"+name)
			}
		}
	}

	rawUsage := ""
	if uf, ok := fl.(cli.DocGenerationFlag); ok {
		rawUsage = uf.GetUsage()
	}

	metavar := flagMetavar(fl, rawUsage, customMetavar)

	allNames := make([]string, 0, len(longNames)+len(shortNames))
	allNames = append(allNames, longNames...)
	allNames = append(allNames, shortNames...)

	var labelBuilder strings.Builder
	labelBuilder.WriteString(strings.Join(allNames, ", "))
	if metavar != "" {
		if labelBuilder.Len() > 0 {
			labelBuilder.WriteString(" ")
		}
		labelBuilder.WriteString(metavar)
	}

	defaultStr, envStr := flagDefaultAndEnv(fl, pathPrefixFilter)
	_, unquoted := unquoteUsage(rawUsage)

	var reqStr string
	if rf, ok := fl.(cli.RequiredFlag); ok && rf.IsRequired() {
		reqStr = " (required)"
	}

	desc := unquoted + reqStr + defaultStr + envStr

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
//
//nolint:cyclop // boundary checking, delimiter detection, and path filtering logic
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
	val := df.GetValue()
	if val == "" {
		return ""
	}

	// Safely strip outer quotes only if it is a single simple quoted literal without internal quotes
	if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' && !strings.Contains(val[1:len(val)-1], `"`) {
		val = val[1 : len(val)-1]
	}

	if val == "" {
		return ""
	}

	// Suppress absolute path defaults matching pathPrefixFilter (e.g. "/home/..."),
	// but do NOT suppress single-character delimiters like "/" or relative paths.
	if pathPrefixFilter != "" && len(val) > len(pathPrefixFilter) && strings.HasPrefix(val, pathPrefixFilter) {
		if strings.Contains(val[len(pathPrefixFilter):], "/") || strings.Contains(val[len(pathPrefixFilter):], "\\") {
			return ""
		}
	}

	return fmt.Sprintf(" (default: %s)", val)
}
