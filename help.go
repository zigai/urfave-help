package urfavehelp

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/urfave/cli/v3"
)

const (
	defaultMinWidth      = 60
	defaultMaxWidth      = 0 // 0 means uncapped: adapt up to terminal width - 1
	defaultWidthFallback = 100
	defaultIndentSpaces  = 2
	defaultGutterGap     = 2
	defaultMinDescWidth  = 20
	defaultPathFilter    = "/"
)

// HelpPrinter is the default HelpPrinterFunc.
var HelpPrinter = NewPrinter()

// Option configures help printer options.
type Option func(*Options)

// Options configures help layout, word wrapping, and formatting.
type Options struct {
	// MinWidth is the lower bound for terminal word wrapping (default: 60).
	MinWidth int

	// MaxWidth is the upper bound for terminal word wrapping (default: 0, uncapped).
	MaxWidth int

	// DefaultWidth is the fallback width when stdout is not a terminal (default: 100).
	DefaultWidth int

	// IndentSpaces is the section indentation (default: 2).
	IndentSpaces int

	// GutterGap is the spacing between labels and descriptions (default: 2).
	GutterGap int

	// MinDescWidth is the minimum width allocated for wrapped descriptions (default: 20).
	MinDescWidth int

	// PathPrefixFilter suppresses flag defaults starting with this prefix (default: "/").
	PathPrefixFilter string

	// CustomMetavar overrides flag placeholder resolution.
	CustomMetavar func(cli.Flag) string
}

// DefaultOptions returns the default formatting configuration.
func DefaultOptions() Options {
	return Options{
		MinWidth:         defaultMinWidth,
		MaxWidth:         defaultMaxWidth,
		DefaultWidth:     defaultWidthFallback,
		IndentSpaces:     defaultIndentSpaces,
		GutterGap:        defaultGutterGap,
		MinDescWidth:     defaultMinDescWidth,
		PathPrefixFilter: defaultPathFilter,
		CustomMetavar:    nil,
	}
}

// WithWidths sets the minimum, maximum, and non-TTY fallback widths for word wrapping.
func WithWidths(minWidth, maxWidth, defaultWidth int) Option {
	return func(o *Options) {
		if minWidth > 0 {
			o.MinWidth = minWidth
		}
		if maxWidth > 0 {
			o.MaxWidth = maxWidth
		}
		if defaultWidth > 0 {
			o.DefaultWidth = defaultWidth
		}
	}
}

// WithGutterGap sets the spacing between labels and descriptions.
func WithGutterGap(gap int) Option {
	return func(o *Options) {
		if gap > 0 {
			o.GutterGap = gap
		}
	}
}

// WithMinDescWidth sets the minimum readable width allocated to wrapped descriptions.
func WithMinDescWidth(width int) Option {
	return func(o *Options) {
		if width > 0 {
			o.MinDescWidth = width
		}
	}
}

// WithPathPrefixFilter sets the prefix used to suppress dynamic path leaks in defaults (e.g. "/").
func WithPathPrefixFilter(prefix string) Option {
	return func(o *Options) {
		o.PathPrefixFilter = prefix
	}
}

// WithMetavarResolver sets a custom resolver for flag placeholder names.
func WithMetavarResolver(fn func(cli.Flag) string) Option {
	return func(o *Options) {
		o.CustomMetavar = fn
	}
}

// Install registers the custom help formatter with urfave/cli.
func Install(opts ...Option) {
	cli.HelpPrinter = NewPrinter(opts...)
}

// NewPrinter creates a cli.HelpPrinterFunc configured with the given options.
func NewPrinter(opts ...Option) cli.HelpPrinterFunc {
	cfg := DefaultOptions()
	for _, opt := range opts {
		opt(&cfg)
	}

	return func(w io.Writer, _ string, data any) {
		cmd, ok := data.(*cli.Command)
		if !ok || cmd == nil {
			return
		}

		PrintHelp(w, cmd, cfg)
	}
}

// PrintHelp formats and writes command help to w.
//
//nolint:gocognit,cyclop // custom help printer layout algorithm
func PrintHelp(w io.Writer, cmd *cli.Command, opts Options) {
	if cmd == nil || w == nil {
		return
	}

	isRoot := cmd.Root() == cmd
	maxWidth := calculateMaxWidth(w, opts)
	prefixGap := opts.IndentSpaces + opts.GutterGap

	// 1. Description on line 1
	desc := cmd.Description
	if desc == "" {
		desc = cmd.Usage
	}
	if desc != "" {
		if strings.Contains(desc, "\n") {
			_, _ = fmt.Fprintln(w, desc)
		} else {
			for _, line := range wrapWords(capitalizeFirst(desc), maxWidth) {
				_, _ = fmt.Fprintln(w, line)
			}
		}

		_, _ = fmt.Fprintln(w)
	}

	// 2. Usage syntax line
	_, _ = fmt.Fprintln(w, "Usage:")
	commands := cmd.VisibleCommands()

	usageLine := strings.Repeat(" ", opts.IndentSpaces) + cmd.FullName()
	if len(commands) > 0 {
		usageLine += " [command]"
	}

	if cmd.ArgsUsage != "" {
		usageLine += " " + cmd.ArgsUsage
	}

	hasFlags := len(cmd.VisibleFlags()) > 0 || len(cmd.VisiblePersistentFlags()) > 0
	if hasFlags {
		usageLine += " [flags]"
	}

	_, _ = fmt.Fprintln(w, usageLine)
	_, _ = fmt.Fprintln(w)

	// Collect items
	args := getArgs(cmd)

	var localFlags []flagItem
	var globalFlags []flagItem

	if isRoot {
		for _, fl := range cmd.VisibleFlags() {
			localFlags = append(localFlags, formatFlagItem(fl, opts.PathPrefixFilter, opts.CustomMetavar))
		}
	} else {
		for _, fl := range cmd.VisibleFlags() {
			names := fl.Names()
			if len(names) > 0 && slices.Contains(names, "help") {
				continue // Deduplicate --help: rendered under Global Flags
			}
			localFlags = append(localFlags, formatFlagItem(fl, opts.PathPrefixFilter, opts.CustomMetavar))
		}

		for _, fl := range cmd.VisiblePersistentFlags() {
			globalFlags = append(globalFlags, formatFlagItem(fl, opts.PathPrefixFilter, opts.CustomMetavar))
		}
	}
	// Calculate unified gutter across args, localFlags, and globalFlags
	gutter := 0
	for _, a := range args {
		gutter = max(gutter, len(a.Name))
	}

	for _, fl := range localFlags {
		gutter = max(gutter, len(fl.label))
	}

	for _, fl := range globalFlags {
		gutter = max(gutter, len(fl.label))
	}
	// 3. Arguments section
	if len(args) > 0 {
		_, _ = fmt.Fprintln(w, "Arguments:")
		for _, a := range args {
			printPaddedEntry(w, a.Name, a.Desc, gutter, maxWidth, prefixGap, opts.MinDescWidth)
		}
		_, _ = fmt.Fprintln(w)
	}

	// 4. Commands section
	if len(commands) > 0 {
		_, _ = fmt.Fprintln(w, "Commands:")
		cmdGutter := 0
		for _, c := range commands {
			names := strings.Join(c.Names(), ", ")
			cmdGutter = max(cmdGutter, len(names))
		}

		for _, c := range commands {
			names := strings.Join(c.Names(), ", ")
			printPaddedEntry(w, names, c.Usage, cmdGutter, maxWidth, prefixGap, opts.MinDescWidth)
		}

		_, _ = fmt.Fprintln(w)
	}
	// 5. Flags section
	if len(localFlags) > 0 {
		_, _ = fmt.Fprintln(w, "Flags:")
		for _, fl := range localFlags {
			printPaddedEntry(w, fl.label, fl.desc, gutter, maxWidth, prefixGap, opts.MinDescWidth)
		}
		_, _ = fmt.Fprintln(w)
	}

	// 6. Global Flags section
	if len(globalFlags) > 0 {
		header := "Global Flags:"
		if isRoot {
			header = "Flags:"
		}
		_, _ = fmt.Fprintln(w, header)
		for _, fl := range globalFlags {
			printPaddedEntry(w, fl.label, fl.desc, gutter, maxWidth, prefixGap, opts.MinDescWidth)
		}
		_, _ = fmt.Fprintln(w)
	}

	// 7. Footer for commands with subcommands
	if len(commands) > 0 {
		_, _ = fmt.Fprintf(w, "Use %q for more information about a command.\n", cmd.FullName()+" [command] --help")
	}
}

func calculateMaxWidth(w io.Writer, opts Options) int {
	if tw := TerminalWidth(w); tw > 0 {
		width := tw - 1
		if opts.MinWidth > 0 && width < opts.MinWidth {
			width = opts.MinWidth
		}
		if opts.MaxWidth > 0 && width > opts.MaxWidth {
			width = opts.MaxWidth
		}
		return width
	}
	if opts.MaxWidth > 0 && opts.DefaultWidth > opts.MaxWidth {
		return opts.MaxWidth
	}
	return opts.DefaultWidth
}
