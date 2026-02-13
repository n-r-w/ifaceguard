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

type packageError struct {
	pos string
	msg string
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

	graph, pkgsExitCode, err := analyzePackages([]*analysis.Analyzer{newAnalyzer}, args, opts.Stderr)
	if err != nil {
		_, _ = fmt.Fprintln(opts.Stderr, err)
		return 1
	}
	if graph == nil {
		return pkgsExitCode
	}

	return outputResults(graph, pkgsExitCode, flags, opts)
}

func analyzePackages(
	analyzers []*analysis.Analyzer,
	args []string,
	stderr io.Writer,
) (*checker.Graph, int, error) {
	allSyntax := needFacts(analyzers)
	initial, err := loadPackages(args, true, allSyntax)
	if err != nil {
		return nil, 0, err
	}

	if n := printPackageErrors(stderr, initial); n > 0 {
		return nil, 1, nil
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

	return graph, 0, nil
}

func printPackageErrors(stderr io.Writer, initial []*packages.Package) int {
	if stderr == nil {
		stderr = os.Stderr
	}

	indices := make(map[string]int)
	uniqueErrors := make([]packageError, 0)

	for _, pkg := range initial {
		for _, pkgErr := range pkg.Errors {
			records := expandPackageError(strings.TrimSpace(pkgErr.Pos), strings.TrimSpace(pkgErr.Msg))
			for _, record := range records {
				key := packageErrorKey(record)
				if idx, ok := indices[key]; ok {
					if shouldPreferPosition(record.pos, uniqueErrors[idx].pos) {
						uniqueErrors[idx].pos = record.pos
					}
					continue
				}
				indices[key] = len(uniqueErrors)
				uniqueErrors = append(uniqueErrors, record)
			}
		}
	}

	for _, pkgErr := range uniqueErrors {
		if pkgErr.pos == "" || pkgErr.pos == "-" {
			_, _ = fmt.Fprintln(stderr, pkgErr.msg)
			continue
		}
		_, _ = fmt.Fprintf(stderr, "%s: %s\n", pkgErr.pos, pkgErr.msg)
	}

	return len(uniqueErrors)
}

func packageErrorKey(pkgErr packageError) string {
	return normalizeErrorPos(pkgErr.pos) + "|" + pkgErr.msg
}

func normalizeErrorPos(pos string) string {
	if pos == "" || pos == "-" {
		return pos
	}

	pathPart, suffix, ok := splitPosition(pos)
	if !ok || pathPart == "" || pathPart == "-" {
		return pos
	}

	cleanPath := filepath.Clean(pathPart)
	if !filepath.IsAbs(cleanPath) {
		absPath, err := filepath.Abs(cleanPath)
		if err == nil {
			cleanPath = absPath
		}
	}
	if resolvedPath, err := filepath.EvalSymlinks(cleanPath); err == nil {
		cleanPath = resolvedPath
	}

	return cleanPath + suffix
}

func splitPosition(pos string) (pathPart, suffix string, ok bool) {
	lineEnd := len(pos) - 1
	for lineEnd >= 0 && pos[lineEnd] >= '0' && pos[lineEnd] <= '9' {
		lineEnd--
	}
	if lineEnd == len(pos)-1 {
		return pos, "", false
	}
	if lineEnd < 0 || pos[lineEnd] != ':' {
		return pos, "", false
	}

	lineStart := lineEnd - 1
	for lineStart >= 0 && pos[lineStart] >= '0' && pos[lineStart] <= '9' {
		lineStart--
	}
	if lineStart >= 0 && pos[lineStart] == ':' && lineStart != lineEnd-1 {
		return pos[:lineStart], pos[lineStart:], true
	}

	return pos[:lineEnd], pos[lineEnd:], true
}

func shouldPreferPosition(candidate, current string) bool {
	if candidate == "" {
		return false
	}
	if current == "" {
		return true
	}
	return len(candidate) < len(current)
}

func expandPackageError(pos, msg string) []packageError {
	if pos != "" && pos != "-" {
		return []packageError{{pos: pos, msg: msg}}
	}

	lines := strings.Split(msg, "\n")
	records := make([]packageError, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		linePos, lineMsg, ok := splitMessagePosition(line)
		if ok {
			records = append(records, packageError{pos: linePos, msg: lineMsg})
			continue
		}

		if strings.HasPrefix(line, "# ") {
			continue
		}

		records = append(records, packageError{
			pos: "",
			msg: line,
		})
	}

	if len(records) == 0 && msg != "" {
		return []packageError{{pos: pos, msg: msg}}
	}

	return records
}

func splitMessagePosition(msg string) (prefix, suffix string, ok bool) {
	separatorIndex := strings.Index(msg, ": ")
	if separatorIndex <= 0 {
		return "", "", false
	}

	prefix = strings.TrimSpace(msg[:separatorIndex])
	if prefix == "" {
		return "", "", false
	}

	_, _, hasPosition := splitPosition(prefix)
	if !hasPosition {
		return "", "", false
	}

	return prefix, strings.TrimSpace(msg[separatorIndex+2:]), true
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
