// Package driver provides a standalone analysis driver for ifaceguard.
package driver

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/n-r-w/ifaceguard/internal/analyzer"
	"github.com/n-r-w/ifaceguard/internal/config"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/checker"
	"golang.org/x/tools/go/packages"
)

const (
	defaultContextLines      = -1
	diagnosticsExitCodeValue = 3
)

// Options controls execution of the standalone driver.
type Options struct {
	Args   []string
	Stdout io.Writer
	Stderr io.Writer
}

type driverFlags struct {
	configPath   string
	jsonOutput   bool
	contextLines int
	version      bool
}

// Run executes a single analyzer using a standalone driver.
// It returns the exit code that should be used by the caller.
func Run(a *analysis.Analyzer, opts Options) int {
	if a == nil {
		_, _ = fmt.Fprintln(os.Stderr, "nil analyzer")
		return 1
	}

	opts = normalizeOptions(opts)

	if err := validateAnalyzers([]*analysis.Analyzer{a}, opts.Stderr); err != nil {
		return 1
	}

	fs, flags := newFlagSet(a, opts)
	if err := fs.Parse(opts.Args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}

	if handled, exitCode := handleMetaFlags(flags, opts); handled {
		return exitCode
	}

	args := fs.Args()
	if len(args) == 0 {
		args = []string{"./..."}
	}

	cfg, err := loadConfig(flags.configPath)
	if err != nil {
		_, _ = fmt.Fprintln(opts.Stderr, err)
		return 1
	}

	return runAnalysis(cfg, args, flags, opts)
}

func normalizeOptions(opts Options) Options {
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	return opts
}

func validateAnalyzers(analyzers []*analysis.Analyzer, stderr io.Writer) error {
	if err := analysis.Validate(analyzers); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return err
	}
	return nil
}

func newFlagSet(a *analysis.Analyzer, opts Options) (*flag.FlagSet, *driverFlags) {
	flags := &driverFlags{
		configPath:   "",
		jsonOutput:   false,
		contextLines: defaultContextLines,
		version:      false,
	}

	fs := flag.NewFlagSet(a.Name, flag.ContinueOnError)
	fs.SetOutput(opts.Stderr)

	fs.StringVar(&flags.configPath, "config", "", "path to ifaceguard config file; if empty, defaults are used")
	fs.BoolVar(&flags.jsonOutput, "json", false, "emit JSON output")
	fs.IntVar(&flags.contextLines, "context", flags.contextLines, "context lines to show; -1 disables source context")
	fs.BoolVar(&flags.version, "version", false, "print version and exit")

	fs.Usage = func() {
		paras := strings.Split(a.Doc, "\n\n")
		_, _ = fmt.Fprintf(opts.Stderr, "%s: %s\n\n", a.Name, paras[0])
		_, _ = fmt.Fprintf(opts.Stderr, "Usage: %s [-flag] [package]\n\n", a.Name)
		if len(paras) > 1 {
			_, _ = fmt.Fprintln(opts.Stderr, strings.Join(paras[1:], "\n\n"))
		}
		_, _ = fmt.Fprintln(opts.Stderr, "\nFlags:")
		fs.PrintDefaults()
	}

	return fs, flags
}

func handleMetaFlags(flags *driverFlags, opts Options) (bool, int) {
	if flags.version {
		if err := printVersion(opts.Stdout); err != nil {
			_, _ = fmt.Fprintln(opts.Stderr, err)
			return true, 1
		}
		return true, 0
	}

	return false, 0
}

func runAnalysis(
	cfg config.Config,
	args []string,
	flags *driverFlags,
	opts Options,
) int {
	newAnalyzer, err := configAnalyzer(cfg)
	if err != nil {
		_, _ = fmt.Fprintln(opts.Stderr, err)
		return 1
	}

	graph, pkgsExitCode, err := analyzePackages([]*analysis.Analyzer{newAnalyzer}, args)
	if err != nil {
		_, _ = fmt.Fprintln(opts.Stderr, err)
		return 1
	}

	return outputResults(graph, pkgsExitCode, flags, opts)
}

func analyzePackages(
	analyzers []*analysis.Analyzer,
	args []string,
) (*checker.Graph, int, error) {
	allSyntax := needFacts(analyzers)
	initial, err := loadPackages(args, true, allSyntax)
	if err != nil {
		return nil, 0, err
	}

	pkgsExitCode := 0
	if n := packages.PrintErrors(initial); n > 0 {
		pkgsExitCode = 1
	}

	checkerOpts := &checker.Options{
		SanityCheck: false,
		Sequential:  false,
		FactLog:     nil,
	}

	graph, err := checker.Analyze(analyzers, initial, checkerOpts)
	if err != nil {
		return nil, 0, err
	}

	return graph, pkgsExitCode, nil
}

func outputResults(
	graph *checker.Graph,
	pkgsExitCode int,
	flags *driverFlags,
	opts Options,
) int {
	if flags.jsonOutput {
		if err := graph.PrintJSON(opts.Stdout); err != nil {
			_, _ = fmt.Fprintln(opts.Stderr, err)
			return 1
		}
		return pkgsExitCode
	}

	if err := graph.PrintText(opts.Stderr, flags.contextLines); err != nil {
		_, _ = fmt.Fprintln(opts.Stderr, err)
		return 1
	}

	diagExitCode := diagnosticsExitCode(graph)
	if diagExitCode != 0 {
		return diagExitCode
	}
	return pkgsExitCode
}

func loadConfig(path string) (config.Config, error) {
	if path == "" {
		return config.Default(), nil
	}
	return config.ParseFromYAMLFile(path)
}

func configAnalyzer(cfg config.Config) (*analysis.Analyzer, error) {
	return analyzer.New(cfg)
}

func printVersion(out io.Writer) error {
	info, ok := debug.ReadBuildInfo()
	name := filepath.Base(os.Args[0])
	if !ok {
		_, err := fmt.Fprintf(out, "%s version unknown\n", name)
		return err
	}

	version := info.Main.Version
	if version == "" || version == "(devel)" {
		version = "devel"
	}

	_, err := fmt.Fprintf(out, "%s version %s\n", name, version)
	return err
}

func loadPackages(patterns []string, includeTests, allSyntax bool) ([]*packages.Package, error) {
	mode := packages.LoadSyntax
	if allSyntax {
		mode = packages.LoadAllSyntax
	}
	mode |= packages.NeedModule
	conf := packages.Config{
		Mode:       mode,
		Tests:      includeTests,
		Context:    nil,
		Logf:       nil,
		Dir:        "",
		Env:        nil,
		BuildFlags: nil,
		Fset:       nil,
		ParseFile:  nil,
		Overlay:    nil,
	}
	pkgs, err := packages.Load(&conf, patterns...)
	if err == nil && len(pkgs) == 0 {
		err = fmt.Errorf("%s matched no packages", strings.Join(patterns, " "))
	}
	return pkgs, err
}

func needFacts(analyzers []*analysis.Analyzer) bool {
	seen := make(map[*analysis.Analyzer]bool)
	var q []*analysis.Analyzer
	q = append(q, analyzers...)
	for len(q) > 0 {
		a := q[0]
		q = q[1:]
		if !seen[a] {
			seen[a] = true
			if len(a.FactTypes) > 0 {
				return true
			}
			q = append(q, a.Requires...)
		}
	}
	return false
}

func diagnosticsExitCode(graph *checker.Graph) int {
	var numErrors int
	var rootDiags int
	graph.All()(func(act *checker.Action) bool {
		if act.Err != nil {
			numErrors++
			return true
		}
		if act.IsRoot {
			rootDiags += len(act.Diagnostics)
		}
		return true
	})
	if numErrors > 0 {
		return 1
	}
	if rootDiags > 0 {
		return diagnosticsExitCodeValue
	}
	return 0
}
