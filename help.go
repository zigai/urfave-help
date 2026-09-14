package urfavehelp

import (
	"fmt"
	"io"
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
//nolint:gocognit,cyclop,maintidx,nestif // custom help printer layout algorithm
func PrintHelp(w io.Writer, cmd *cli.Command, opts Options) {
	if cmd == nil || w == nil {
		return
	}

	isRoot := cmd.Root() == cmd
	maxWidth := calculateMaxWidth(w, opts)
	prefixGap := opts.IndentSpaces + opts.GutterGap

	// 1. Description / Summary on line 1
	usage := cmd.Usage
	if usage == "A new cli application" {
		usage = ""
	}
	desc := cmd.Description
	if usage != "" {
		if strings.Contains(usage, "\n") {
			_, _ = fmt.Fprintln(w, usage)
		} else {
			for _, line := range wrapWords(usage, maxWidth) {
				_, _ = fmt.Fprintln(w, line)
			}
		}
		_, _ = fmt.Fprintln(w)
		if desc != "" && desc != usage {
			if strings.Contains(desc, "\n") {
				_, _ = fmt.Fprintln(w, desc)
			} else {
				for _, line := range wrapWords(desc, maxWidth) {
					_, _ = fmt.Fprintln(w, line)
				}
			}
			_, _ = fmt.Fprintln(w)
		}
	} else if desc != "" {
		if strings.Contains(desc, "\n") {
			_, _ = fmt.Fprintln(w, desc)
		} else {
			for _, line := range wrapWords(desc, maxWidth) {
				_, _ = fmt.Fprintln(w, line)
			}
		}
		_, _ = fmt.Fprintln(w)
	}

	// 2. Usage syntax line
	_, _ = fmt.Fprintln(w, "Usage:")
	if cmd.UsageText != "" {
		for l := range strings.SplitSeq(cmd.UsageText, "\n") {
			if strings.HasPrefix(l, " ") || strings.HasPrefix(l, "\t") {
				_, _ = fmt.Fprintln(w, l)
			} else {
				_, _ = fmt.Fprintln(w, strings.Repeat(" ", opts.IndentSpaces)+l)
			}
		}
		_, _ = fmt.Fprintln(w)
	} else {
		usageLine := strings.Repeat(" ", opts.IndentSpaces) + cmd.FullName()
		commands := cmd.VisibleCommands()
		if len(commands) > 0 {
			usageLine += " [command]"
		}

		hasFlags := len(cmd.VisibleFlags()) > 0 || len(cmd.VisiblePersistentFlags()) > 0
		if hasFlags {
			usageLine += " [flags]"
		}

		if cmd.ArgsUsage != "" {
			usageLine += " " + cmd.ArgsUsage
		} else if len(cmd.Arguments) > 0 {
			usageLine += " [arguments...]"
		}

		_, _ = fmt.Fprintln(w, usageLine)
		_, _ = fmt.Fprintln(w)
	}

	// Collect items
	args := getArgs(cmd)

	// Collect persistent flag names to avoid duplicates in subcommands
	persistentNames := make(map[string]bool)
	for _, pfl := range cmd.VisiblePersistentFlags() {
		for _, name := range pfl.Names() {
			persistentNames[name] = true
		}
	}

	var localFlags []flagItem
	var globalFlags []flagItem

	if isRoot {
		for _, fl := range cmd.VisibleFlags() {
			localFlags = append(localFlags, formatFlagItem(fl, opts.PathPrefixFilter, opts.CustomMetavar))
		}
	} else {
		for _, fl := range cmd.VisibleFlags() {
			names := fl.Names()
			isPersistent := false
			for _, n := range names {
				if persistentNames[n] {
					isPersistent = true
					break
				}
			}
			if isPersistent {
				continue
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
	commands := cmd.VisibleCommands()
	if len(commands) > 0 {
		_, _ = fmt.Fprintln(w, "Commands:")
		cmdGutter := 0
		for _, c := range commands {
			names := strings.Join(c.Names(), ", ")
			cmdGutter = max(cmdGutter, len(names))
		}

		// Group visible commands by category safely without relying on uninitialized internal state
		categories := make(map[string][]*cli.Command)
		var catOrder []string
		var uncategorized []*cli.Command

		for _, c := range commands {
			cat := strings.TrimSpace(c.Category)
			if cat == "" {
				uncategorized = append(uncategorized, c)
			} else {
				if _, exists := categories[cat]; !exists {
					catOrder = append(catOrder, cat)
				}
				categories[cat] = append(categories[cat], c)
			}
		}

		if len(catOrder) > 0 {
			// Uncategorized commands first
			for _, c := range uncategorized {
				names := strings.Join(c.Names(), ", ")
				printPaddedEntry(w, names, c.Usage, cmdGutter, maxWidth, prefixGap, opts.MinDescWidth)
			}

			// Then each named category
			for _, catName := range catOrder {
				catCmds := categories[catName]
				if len(catCmds) == 0 {
					continue
				}
				if len(uncategorized) > 0 || catName != catOrder[0] {
					_, _ = fmt.Fprintln(w)
				}
				_, _ = fmt.Fprintf(w, "%s%s:\n", strings.Repeat(" ", opts.IndentSpaces), catName)
				subIndent := opts.IndentSpaces + opts.IndentSpaces
				for _, c := range catCmds {
					names := strings.Join(c.Names(), ", ")
					printPaddedEntryIndent(w, names, c.Usage, cmdGutter, maxWidth, subIndent, opts.GutterGap, opts.MinDescWidth)
				}
			}
		} else {
			for _, c := range commands {
				names := strings.Join(c.Names(), ", ")
				printPaddedEntry(w, names, c.Usage, cmdGutter, maxWidth, prefixGap, opts.MinDescWidth)
			}
		}

		_, _ = fmt.Fprintln(w)
	}

	// 5. Flags section
	if len(localFlags) > 0 {
		flagCategories := cmd.VisibleFlagCategories()
		var namedFlagCats []cli.VisibleFlagCategory
		for _, fc := range flagCategories {
			if fc.Name() != "" && len(fc.Flags()) > 0 {
				namedFlagCats = append(namedFlagCats, fc)
			}
		}

		if len(namedFlagCats) > 0 {
			categorizedFlagSet := make(map[cli.Flag]bool)
			for _, fc := range namedFlagCats {
				for _, fl := range fc.Flags() {
					categorizedFlagSet[fl] = true
				}
			}

			var uncategorizedFlags []flagItem
			for _, fl := range cmd.VisibleFlags() {
				if !categorizedFlagSet[fl] {
					names := fl.Names()
					isPersistent := false
					if !isRoot {
						for _, n := range names {
							if persistentNames[n] {
								isPersistent = true
								break
							}
						}
					}
					if !isPersistent {
						uncategorizedFlags = append(uncategorizedFlags, formatFlagItem(fl, opts.PathPrefixFilter, opts.CustomMetavar))
					}
				}
			}

			if len(uncategorizedFlags) > 0 {
				_, _ = fmt.Fprintln(w, "Flags:")
				for _, fl := range uncategorizedFlags {
					printPaddedEntry(w, fl.label, fl.desc, gutter, maxWidth, prefixGap, opts.MinDescWidth)
				}
			}

			for _, fc := range namedFlagCats {
				var catFlags []flagItem
				for _, fl := range fc.Flags() {
					names := fl.Names()
					isPersistent := false
					if !isRoot {
						for _, n := range names {
							if persistentNames[n] {
								isPersistent = true
								break
							}
						}
					}
					if !isPersistent {
						catFlags = append(catFlags, formatFlagItem(fl, opts.PathPrefixFilter, opts.CustomMetavar))
					}
				}
				if len(catFlags) > 0 {
					if len(uncategorizedFlags) > 0 || fc != namedFlagCats[0] {
						_, _ = fmt.Fprintln(w)
					}
					_, _ = fmt.Fprintf(w, "%s:\n", fc.Name())
					for _, fl := range catFlags {
						printPaddedEntry(w, fl.label, fl.desc, gutter, maxWidth, prefixGap, opts.MinDescWidth)
					}
				}
			}
		} else {
			_, _ = fmt.Fprintln(w, "Flags:")
			for _, fl := range localFlags {
				printPaddedEntry(w, fl.label, fl.desc, gutter, maxWidth, prefixGap, opts.MinDescWidth)
			}
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

	// 8. Authors (clean & compact)
	if len(cmd.Authors) > 0 {
		_, _ = fmt.Fprintln(w)
		if len(cmd.Authors) == 1 {
			_, _ = fmt.Fprintf(w, "Author: %s\n", cmd.Authors[0])
		} else {
			_, _ = fmt.Fprintln(w, "Authors:")
			for _, a := range cmd.Authors {
				_, _ = fmt.Fprintf(w, "%s%s\n", strings.Repeat(" ", opts.IndentSpaces), a)
			}
		}
	}

	// 9. Copyright (clean & compact)
	if cmd.Copyright != "" {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintf(w, "Copyright: %s\n", cmd.Copyright)
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
