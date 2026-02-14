// Package analyzer provides the ifaceguard analysis.Analyzer for detecting
// interface architecture violations in Go code.
package analyzer

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/n-r-w/ifaceguard/internal/config"
	"github.com/n-r-w/ifaceguard/internal/typeutil"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"
)

// AnalyzerName is the identifier used for the analyzer in reports and configuration.
const (
	AnalyzerName     = "ifaceguard"
	ownershipID      = "IFG001-OWNERSHIP"
	assertionsID     = "IFG002-ASSERTION-PLACEMENT"
	assertionsMissID = "IFG003-ASSERTION-MISSING"
	assertionsBypass = "IFG004-ASSERTION-BYPASS"
	minCoImportCount = 2
)

// analyzerState holds configuration and dependencies for the analyzer.
type analyzerState struct {
	cfg config.Config

	globalDataOnce   sync.Once
	globalInterfaces []*interfaceRefInfo
	globalCoImports  map[string]map[string]struct{}
	globalDataErr    error
}

// New creates a new ifaceguard analyzer.
// The cfg parameter provides rule configuration; use config.Default() for defaults.
func New(cfg config.Config) (*analysis.Analyzer, error) {
	return newAnalyzer(cfg)
}

func newAnalyzer(cfg config.Config) (*analysis.Analyzer, error) {
	//nolint:exhaustruct // global cache fields are optional and initialized lazily
	a := &analyzerState{cfg: cfg}
	//nolint:exhaustruct // zero values are appropriate for optional fields
	return &analysis.Analyzer{
		Name: AnalyzerName,
		Doc:  "checks architectural properties of interface usage in Go code",
		Run:  a.run,
	}, nil
}

// run is the analyzer entry point.
func (a *analyzerState) run(pass *analysis.Pass) (any, error) {
	effective := a.cfg.ResolveEffective()

	// Compile effective config to get regex patterns ready.
	compiled, err := effective.Compile()
	if err != nil {
		return nil, err
	}

	// Assertions rule: check compile-time assertion placement.
	if compiled.Assertions.Enabled {
		if err := a.checkAssertions(pass, compiled.Ownership, compiled.Assertions, compiled.Exclude); err != nil {
			return nil, err
		}
	}

	// Ownership rule: check interface ownership violations.
	// Note: compiled.Ownership.ContractScope is non-nil after effective resolution.
	if compiled.Ownership.Enabled {
		switch *compiled.Ownership.ContractScope {
		case config.ContractScopeExportedOutput:
			a.checkOwnershipExportedOutput(pass, compiled.Ownership, compiled.Exclude)
		case config.ContractScopeAnyExported:
			a.checkOwnershipAnyExported(pass, compiled.Ownership, compiled.Exclude)
		}
	}

	return nil, nil //nolint:nilnil // analyzer returns no facts
}

// checkAssertions scans var declarations for compile-time assertions
// and reports violations when assertions are not in the implementation type's package.
// When cfg.ScanFunctionBodies is true, also scans inside function/method bodies.
func (a *analyzerState) checkAssertions(
	pass *analysis.Pass,
	ownership config.CompiledOwnershipConfig,
	assertionsCfg config.CompiledAssertionsConfig,
	exclude config.CompiledExcludeConfig,
) error {
	var assertions []assertionInfo
	for _, file := range pass.Files {
		if isFileExcluded(pass.Fset, file.Pos(), exclude) {
			continue
		}

		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				a.checkVarDecl(pass, assertionsCfg, exclude, d, &assertions)
			case *ast.FuncDecl:
				if assertionsCfg.ScanFunctionBodies && d.Body != nil {
					a.scanFuncBodyForAssertions(pass, assertionsCfg, exclude, d.Body, &assertions)
				}
			}
		}
	}

	if assertionsCfg.RequireAssertions {
		return a.checkAssertionsRequireAssertions(pass, ownership, assertionsCfg, exclude, assertions)
	}

	return nil
}

func (a *analyzerState) checkVarDecl(
	pass *analysis.Pass,
	cfg config.CompiledAssertionsConfig,
	exclude config.CompiledExcludeConfig,
	decl *ast.GenDecl,
	assertions *[]assertionInfo,
) {
	if decl.Tok != token.VAR {
		return
	}
	for _, spec := range decl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		infos := a.checkAssertionSpec(pass, cfg, exclude, valueSpec)
		if assertions != nil {
			*assertions = append(*assertions, infos...)
		}
	}
}

// scanFuncBodyForAssertions scans a function body for var declarations that may be assertions.
func (a *analyzerState) scanFuncBodyForAssertions(
	pass *analysis.Pass,
	cfg config.CompiledAssertionsConfig,
	exclude config.CompiledExcludeConfig,
	body *ast.BlockStmt,
	assertions *[]assertionInfo,
) {
	ast.Inspect(body, func(n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.DeclStmt:
			genDecl, ok := stmt.Decl.(*ast.GenDecl)
			if !ok {
				return true
			}
			a.checkVarDecl(pass, cfg, exclude, genDecl, assertions)
		case *ast.AssignStmt:
			valueSpec := valueSpecFromAssertionAssign(stmt)
			if valueSpec == nil {
				return true
			}
			infos := a.checkAssertionSpec(pass, cfg, exclude, valueSpec)
			if assertions != nil {
				*assertions = append(*assertions, infos...)
			}
		}
		return true
	})
}

func valueSpecFromAssertionAssign(stmt *ast.AssignStmt) *ast.ValueSpec {
	if stmt.Tok != token.ASSIGN {
		return nil
	}
	if len(stmt.Lhs) != 1 || len(stmt.Rhs) != 1 {
		return nil
	}
	lhsIdent, ok := stmt.Lhs[0].(*ast.Ident)
	if !ok || lhsIdent.Name != "_" {
		return nil
	}
	return &ast.ValueSpec{
		Doc:     nil,
		Names:   []*ast.Ident{lhsIdent},
		Type:    nil,
		Values:  []ast.Expr{stmt.Rhs[0]},
		Comment: nil,
	}
}

// checkAssertionSpec checks a single ValueSpec for compile-time assertion violations.
func (a *analyzerState) checkAssertionSpec(
	pass *analysis.Pass,
	cfg config.CompiledAssertionsConfig,
	exclude config.CompiledExcludeConfig,
	spec *ast.ValueSpec,
) []assertionInfo {
	if !hasBlankIdentifier(spec.Names) {
		return nil
	}

	// Determine LHS interface type based on spec form.
	var lhsType types.Type
	if spec.Type != nil {
		// Explicit type annotation: var _ I = expr
		lhsType = pass.TypesInfo.TypeOf(spec.Type)
	} else if cfg.AcceptConversionOnlyForm {
		// Conversion-only form: var _ = I(expr) - requires flag.
		lhsType = extractConversionTargetInterface(pass.TypesInfo, spec)
	}

	if lhsType == nil {
		return nil
	}

	// Check if underlying type is interface.
	underlying := lhsType.Underlying()
	if _, ok := underlying.(*types.Interface); !ok {
		return nil
	}

	if lhsNamed, ok := lhsType.(*types.Named); ok {
		if fullName, ok := fullNamedTypeName(lhsNamed); ok && isTypeExcluded(exclude, fullName) {
			return nil
		}
	}

	infos := make([]assertionInfo, 0, len(spec.Names))

	// Process each init expression paired with blank identifier.
	for i, name := range spec.Names {
		if name.Name != "_" {
			continue
		}
		if i >= len(spec.Values) {
			continue
		}

		initExpr := spec.Values[i]
		implType := extractImplementationType(pass.TypesInfo, initExpr)
		if implType == nil {
			continue
		}
		if fullName, ok := fullNamedTypeName(implType); ok && isTypeExcluded(exclude, fullName) {
			continue
		}

		// Get the package where implementation type is declared.
		implPkg := implType.Obj().Pkg()
		if implPkg == nil {
			continue
		}

		currentPkg := pass.Pkg
		if implPkg.Path() == currentPkg.Path() {
			if bypassKind, isBypass := assertionBypassKind(pass, cfg, exclude, lhsType); isBypass {
				msg := fmt.Sprintf(
					"%s: assertion for type %q uses %s and can bypass explicit contract assertion. "+
						"Use named external contract form: \"var _ pkg.Interface = (*%s)(nil)\"",
					assertionsBypass,
					implType.Obj().Name(),
					bypassKind,
					implType.Obj().Name(),
				)
				reportDiagnostic(pass, name.Pos(), assertionsBypass, msg)
				continue
			}
		}

		infos = append(infos, assertionInfo{
			implType: implType,
			lhsType:  lhsType,
		})

		// B3: current package must equal implementation package.
		if implPkg.Path() == currentPkg.Path() {
			continue // correct placement, no violation
		}

		// B4: check wiringpackages exclusion.
		if matchesAnyPattern(currentPkg.Path(), cfg.WiringPackages) {
			continue
		}

		// Report violation at the variable declaration position.
		msg := fmt.Sprintf(
			"%s: compile-time assertion for type %q should be in package %q. "+
				"Current package: %q",
			assertionsID,
			implType.Obj().Name(),
			implPkg.Name(),
			currentPkg.Name(),
		)
		reportDiagnostic(pass, name.Pos(), assertionsID, msg)
	}

	return infos
}

// assertionBypassKind reports whether an assertion LHS represents a bypass pattern
// that should not satisfy IFG003 when requireassertions is enabled.
func assertionBypassKind(
	pass *analysis.Pass,
	cfg config.CompiledAssertionsConfig,
	exclude config.CompiledExcludeConfig,
	lhsType types.Type,
) (string, bool) {
	if !cfg.RequireAssertions {
		return "", false
	}
	if !cfg.CheckBypassAssertions {
		return "", false
	}

	switch t := lhsType.(type) {
	case *types.Interface:
		return "anonymous interface", true
	case *types.Named:
		if _, ok := t.Underlying().(*types.Interface); !ok {
			return "", false
		}

		typeName := typeutil.UnaliasTypeName(t.Obj())
		if typeName == nil {
			return "", false
		}

		pkg := typeName.Pkg()
		if pkg == nil || pkg.Path() != pass.Pkg.Path() {
			return "", false
		}
		if typeName.Exported() {
			return "", false
		}

		if !isTypeNameUsedOnlyInAssertions(pass, cfg, exclude, typeName) {
			return "", false
		}

		return fmt.Sprintf("private interface %q used only in assertion", typeName.Name()), true
	default:
		return "", false
	}
}

// isTypeNameUsedOnlyInAssertions reports whether all references to typeName are assertion usages.
func isTypeNameUsedOnlyInAssertions(
	pass *analysis.Pass,
	cfg config.CompiledAssertionsConfig,
	exclude config.CompiledExcludeConfig,
	typeName *types.TypeName,
) bool {
	if pass == nil || pass.TypesInfo == nil || typeName == nil {
		return false
	}

	totalUses := 0
	for ident, obj := range pass.TypesInfo.Uses {
		if obj == typeName {
			if isFileExcluded(pass.Fset, ident.Pos(), exclude) {
				continue
			}
			totalUses++
		}
	}
	if totalUses == 0 {
		return false
	}

	assertionUses := countTypeNameAssertionUses(pass, cfg, exclude, typeName)
	return assertionUses > 0 && assertionUses == totalUses
}

// countTypeNameAssertionUses counts assertion declarations that use typeName as LHS interface.
func countTypeNameAssertionUses(
	pass *analysis.Pass,
	cfg config.CompiledAssertionsConfig,
	exclude config.CompiledExcludeConfig,
	typeName *types.TypeName,
) int {
	count := 0
	for _, file := range pass.Files {
		if file == nil || isFileExcluded(pass.Fset, file.Pos(), exclude) {
			continue
		}

		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				count += countTypeNameAssertionUsesInGenDecl(pass, cfg, exclude, d, typeName)
			case *ast.FuncDecl:
				if cfg.ScanFunctionBodies && d.Body != nil {
					count += countTypeNameAssertionUsesInFuncBody(pass, cfg, exclude, d.Body, typeName)
				}
			}
		}
	}

	return count
}

// countTypeNameAssertionUsesInGenDecl counts assertion usages from a package-level var declaration.
func countTypeNameAssertionUsesInGenDecl(
	pass *analysis.Pass,
	cfg config.CompiledAssertionsConfig,
	exclude config.CompiledExcludeConfig,
	decl *ast.GenDecl,
	typeName *types.TypeName,
) int {
	if decl == nil || decl.Tok != token.VAR {
		return 0
	}

	count := 0
	for _, spec := range decl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		if valueSpecUsesTypeNameInAssertion(pass, cfg, exclude, valueSpec, typeName) {
			count++
		}
	}

	return count
}

// countTypeNameAssertionUsesInFuncBody counts assertion usages from function-body declarations.
func countTypeNameAssertionUsesInFuncBody(
	pass *analysis.Pass,
	cfg config.CompiledAssertionsConfig,
	exclude config.CompiledExcludeConfig,
	body *ast.BlockStmt,
	typeName *types.TypeName,
) int {
	count := 0
	ast.Inspect(body, func(n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.DeclStmt:
			genDecl, ok := stmt.Decl.(*ast.GenDecl)
			if !ok {
				return true
			}
			count += countTypeNameAssertionUsesInGenDecl(pass, cfg, exclude, genDecl, typeName)
		case *ast.AssignStmt:
			valueSpec := valueSpecFromAssertionAssign(stmt)
			if valueSpec == nil {
				return true
			}
			if valueSpecUsesTypeNameInAssertion(pass, cfg, exclude, valueSpec, typeName) {
				count++
			}
		}
		return true
	})

	return count
}

// valueSpecUsesTypeNameInAssertion checks whether spec is a valid assertion using typeName as LHS.
func valueSpecUsesTypeNameInAssertion(
	pass *analysis.Pass,
	cfg config.CompiledAssertionsConfig,
	exclude config.CompiledExcludeConfig,
	spec *ast.ValueSpec,
	typeName *types.TypeName,
) bool {
	if spec == nil || typeName == nil || !hasBlankIdentifier(spec.Names) {
		return false
	}

	var lhsType types.Type
	if spec.Type != nil {
		lhsType = pass.TypesInfo.TypeOf(spec.Type)
	} else if cfg.AcceptConversionOnlyForm {
		lhsType = extractConversionTargetInterface(pass.TypesInfo, spec)
	}
	if lhsType == nil {
		return false
	}

	lhsNamed, ok := lhsType.(*types.Named)
	if !ok {
		return false
	}
	if _, ok := lhsType.Underlying().(*types.Interface); !ok {
		return false
	}

	owner := typeutil.UnaliasTypeName(lhsNamed.Obj())
	if owner == nil || owner != typeName {
		return false
	}

	if fullName, ok := fullNamedTypeName(lhsNamed); ok && isTypeExcluded(exclude, fullName) {
		return false
	}

	for i, name := range spec.Names {
		if name.Name != "_" || i >= len(spec.Values) {
			continue
		}

		implType := extractImplementationType(pass.TypesInfo, spec.Values[i])
		if implType == nil {
			continue
		}
		if fullName, ok := fullNamedTypeName(implType); ok && isTypeExcluded(exclude, fullName) {
			continue
		}

		return true
	}

	return false
}

// hasBlankIdentifier returns true if names contains a blank identifier "_".
func hasBlankIdentifier(names []*ast.Ident) bool {
	for _, name := range names {
		if name.Name == "_" {
			return true
		}
	}
	return false
}

func isFileExcluded(fset *token.FileSet, pos token.Pos, exclude config.CompiledExcludeConfig) bool {
	if len(exclude.Files) == 0 {
		return false
	}
	if !pos.IsValid() {
		return false
	}
	file := fset.File(pos)
	if file == nil {
		return false
	}

	fileName := filepath.ToSlash(file.Name())
	return matchesAnyPattern(fileName, exclude.Files)
}

// extractConversionTargetInterface extracts the target interface type from a conversion-only
// assertion form: var _ = I(expr). Returns nil if not a valid conversion to interface.
func extractConversionTargetInterface(info *types.Info, spec *ast.ValueSpec) types.Type {
	// Conversion-only form requires single blank identifier with single value.
	if len(spec.Names) != 1 || spec.Names[0].Name != "_" || len(spec.Values) != 1 {
		return nil
	}

	// The value must be a call expression (type conversion in Go AST).
	callExpr, ok := spec.Values[0].(*ast.CallExpr)
	if !ok {
		return nil
	}

	// Type conversion must have exactly one argument.
	if len(callExpr.Args) != 1 {
		return nil
	}

	// Verify this is actually a type conversion, not a function call.
	// In a type conversion, the Fun position has IsType() == true.
	tv, ok := info.Types[callExpr.Fun]
	if !ok || !tv.IsType() {
		return nil
	}

	targetType := tv.Type
	if targetType == nil {
		return nil
	}

	// Target must be an interface type.
	if _, ok := targetType.Underlying().(*types.Interface); !ok {
		return nil
	}

	return targetType
}

// extractImplementationType extracts the named implementation type from an init expression.
// Supports forms: (*T)(nil), new(T), T{}, &T{}.
// For conversion-only form I(expr), extracts from the conversion argument.
func extractImplementationType(info *types.Info, expr ast.Expr) *types.Named {
	// Handle type conversion: I(expr) or (*T)(nil).
	if named := extractFromTypeConversion(info, expr); named != nil {
		return named
	}

	return extractFromExprType(info, expr)
}

// extractFromTypeConversion handles type conversion expressions like I(expr) or (*T)(nil).
func extractFromTypeConversion(info *types.Info, expr ast.Expr) *types.Named {
	callExpr, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil
	}

	tv, ok := info.Types[callExpr.Fun]
	if !ok || !tv.IsType() || len(callExpr.Args) != 1 {
		return nil
	}

	convType := tv.Type

	// If converting to interface, recurse into argument.
	if _, isIface := convType.Underlying().(*types.Interface); isIface {
		return extractImplementationType(info, callExpr.Args[0])
	}

	// If converting to pointer type (like (*T)(nil)), extract T from the pointer.
	if ptr, ok := convType.(*types.Pointer); ok {
		if named, ok := ptr.Elem().(*types.Named); ok {
			return named
		}
		return nil
	}

	// If converting to named type directly.
	if named, ok := convType.(*types.Named); ok {
		return named
	}

	return nil
}

// extractFromExprType extracts named type from expression's type.
func extractFromExprType(info *types.Info, expr ast.Expr) *types.Named {
	exprType := info.TypeOf(expr)
	if exprType == nil {
		return nil
	}

	// Handle pointer types.
	if ptr, ok := exprType.(*types.Pointer); ok {
		if named, ok := ptr.Elem().(*types.Named); ok {
			return named
		}
		return nil
	}

	// Handle value types.
	if named, ok := exprType.(*types.Named); ok {
		return named
	}

	return nil
}

// matchesAnyPattern returns true if path matches any of the patterns.
func matchesAnyPattern(path string, patterns []*regexp.Regexp) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(path) {
			return true
		}
	}
	return false
}

func isTypeExcluded(exclude config.CompiledExcludeConfig, fullName string) bool {
	return matchesAnyPattern(fullName, exclude.Types)
}

func fullNamedTypeName(named *types.Named) (string, bool) {
	if named == nil {
		return "", false
	}
	obj := named.Obj()
	if obj == nil {
		return "", false
	}
	pkg := obj.Pkg()
	if pkg == nil {
		return "", false
	}
	return pkg.Path() + "." + obj.Name(), true
}

func moduleRootFromPass(pass *analysis.Pass) (string, error) {
	if pass == nil {
		return "", errors.New("pass is nil")
	}
	for _, file := range pass.Files {
		if file == nil {
			continue
		}
		pos := file.Pos()
		if !pos.IsValid() {
			continue
		}
		f := pass.Fset.File(pos)
		if f == nil || f.Name() == "" {
			continue
		}
		return findModuleRoot(filepath.Dir(f.Name()))
	}
	return "", errors.New("unable to locate module root from package files")
}

func findModuleRoot(startDir string) (string, error) {
	if startDir == "" {
		return "", errors.New("start directory is empty")
	}
	current := filepath.Clean(startDir)
	for {
		candidate := filepath.Join(current, "go.mod")
		if _, err := os.Stat(candidate); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "", fmt.Errorf("go.mod not found starting from %s", startDir)
}

func (a *analyzerState) loadGlobalState(
	pass *analysis.Pass,
	ownership config.CompiledOwnershipConfig,
	exclude config.CompiledExcludeConfig,
) ([]*interfaceRefInfo, map[string]map[string]struct{}, error) {
	a.globalDataOnce.Do(func() {
		a.globalInterfaces, a.globalCoImports, a.globalDataErr = collectGlobalState(pass, ownership, exclude)
	})
	return a.globalInterfaces, a.globalCoImports, a.globalDataErr
}

func collectGlobalState(
	pass *analysis.Pass,
	ownership config.CompiledOwnershipConfig,
	exclude config.CompiledExcludeConfig,
) ([]*interfaceRefInfo, map[string]map[string]struct{}, error) {
	moduleRoot, err := moduleRootFromPass(pass)
	if err != nil {
		return nil, nil, err
	}

	mode := packages.NeedName |
		packages.NeedTypes |
		packages.NeedTypesInfo |
		packages.NeedSyntax |
		packages.NeedFiles |
		packages.NeedImports |
		packages.NeedCompiledGoFiles

	//nolint:exhaustruct // only relevant fields are set for package loading
	pkgs, err := packages.Load(&packages.Config{
		Mode:  mode,
		Dir:   moduleRoot,
		Tests: false,
	}, "./...")
	if err != nil {
		return nil, nil, err
	}

	result := make([]*interfaceRefInfo, 0)
	coImports := collectCoImports(pkgs)
	seen := make(map[string]struct{})
	packages.Visit(pkgs, func(pkg *packages.Package) bool {
		if packageIsSkippable(pkg) {
			return true
		}

		packagePass := buildPackagePass(pkg)
		interfaces := collectPackageInterfaces(packagePass, pkg.Syntax, exclude)
		if len(interfaces) == 0 {
			return true
		}

		contractual := contractualInterfacesForPackage(
			packagePass,
			pkg.Syntax,
			interfaces,
			ownership,
			exclude,
		)
		appendContractualInterfaces(&result, seen, interfaces, contractual, ownership, exclude, packagePass)
		return true
	}, nil)

	return result, coImports, nil
}

func packageIsSkippable(pkg *packages.Package) bool {
	if pkg == nil || pkg.Types == nil || pkg.TypesInfo == nil || pkg.Fset == nil {
		return true
	}
	if pkg.ForTest != "" {
		return true
	}
	if len(pkg.Errors) > 0 {
		return true
	}
	return len(pkg.Syntax) == 0
}

func collectCoImports(pkgs []*packages.Package) map[string]map[string]struct{} {
	result := make(map[string]map[string]struct{})
	packages.Visit(pkgs, func(pkg *packages.Package) bool {
		if pkg == nil || pkg.PkgPath == "" {
			return true
		}
		if pkg.ForTest != "" {
			return true
		}
		if len(pkg.Imports) < minCoImportCount {
			return true
		}

		importPaths := make([]string, 0, len(pkg.Imports))
		for _, imported := range pkg.Imports {
			if imported == nil || imported.PkgPath == "" {
				continue
			}
			importPaths = append(importPaths, imported.PkgPath)
		}
		for i := range importPaths {
			for j := range importPaths[i+1:] {
				left := importPaths[i]
				right := importPaths[i+1+j]
				addCoImport(result, left, right)
				addCoImport(result, right, left)
			}
		}

		return true
	}, nil)

	return result
}

func addCoImport(result map[string]map[string]struct{}, left string, right string) {
	if left == "" || right == "" {
		return
	}
	set, exists := result[left]
	if !exists {
		set = make(map[string]struct{})
		result[left] = set
	}
	set[right] = struct{}{}
}

func buildPackagePass(pkg *packages.Package) *analysis.Pass {
	//nolint:exhaustruct // only required fields are set for helper functions
	return &analysis.Pass{
		TypesInfo: pkg.TypesInfo,
		Fset:      pkg.Fset,
	}
}

func collectPackageInterfaces(
	pass *analysis.Pass,
	files []*ast.File,
	exclude config.CompiledExcludeConfig,
) []*interfaceInfo {
	var interfaces []*interfaceInfo
	var implCandidates []*types.Named
	for _, file := range files {
		if file == nil {
			continue
		}
		if isFileExcluded(pass.Fset, file.Pos(), exclude) {
			continue
		}
		collectTypesFromFile(pass, file, &interfaces, &implCandidates)
	}
	return interfaces
}

func contractualInterfacesForPackage(
	pass *analysis.Pass,
	files []*ast.File,
	interfaces []*interfaceInfo,
	ownership config.CompiledOwnershipConfig,
	exclude config.CompiledExcludeConfig,
) map[*types.TypeName]bool {
	contractual := make(map[*types.TypeName]bool)
	if ownership.ContractScope == nil {
		return contractual
	}

	switch *ownership.ContractScope {
	case config.ContractScopeAnyExported:
		for _, ifaceInfo := range interfaces {
			if ifaceInfo.typeName.Exported() {
				contractual[ifaceInfo.typeName] = true
			}
		}
	case config.ContractScopeExportedOutput:
		for _, file := range files {
			if file == nil {
				continue
			}
			if isFileExcluded(pass.Fset, file.Pos(), exclude) {
				continue
			}
			collectExportedOutputSurfaces(pass, file, interfaces, contractual, ownership.SkipIfUsedAsInput)
		}
	}

	return contractual
}

func appendContractualInterfaces(
	result *[]*interfaceRefInfo,
	seen map[string]struct{},
	interfaces []*interfaceInfo,
	contractual map[*types.TypeName]bool,
	ownership config.CompiledOwnershipConfig,
	exclude config.CompiledExcludeConfig,
	pass *analysis.Pass,
) {
	if len(contractual) == 0 {
		return
	}
	for _, ifaceInfo := range interfaces {
		if !ifaceInfo.typeName.Exported() {
			continue
		}
		appearsInOutput, found := contractual[ifaceInfo.typeName]
		if !found || !appearsInOutput {
			continue
		}

		ownerPkg := ifaceInfo.typeName.Pkg()
		if ownerPkg == nil {
			continue
		}
		fullName := ownerPkg.Path() + "." + ifaceInfo.typeName.Name()
		if _, exists := seen[fullName]; exists {
			continue
		}
		if isTypeExcluded(exclude, fullName) {
			continue
		}
		if isOwnershipInterfaceIgnored(ownership, fullName, ifaceInfo.named) {
			continue
		}
		if isFileExcluded(pass.Fset, ifaceInfo.pos, exclude) {
			continue
		}

		seen[fullName] = struct{}{}
		*result = append(*result, &interfaceRefInfo{
			typeName: ifaceInfo.typeName,
			named:    ifaceInfo.named,
			iface:    ifaceInfo.iface,
			fullName: fullName,
		})
	}
}

func interfaceIsRelevantForImpl(
	pass *analysis.Pass,
	ifacePkgPath string,
	coImports map[string]map[string]struct{},
) bool {
	if pass == nil || pass.Pkg == nil || ifacePkgPath == "" {
		return false
	}
	if importsPackage(pass.Pkg, ifacePkgPath) {
		return true
	}
	implPath := pass.Pkg.Path()
	if implPath == "" {
		return false
	}
	if coImports == nil {
		return false
	}
	ifaceSet, ok := coImports[implPath]
	if !ok {
		return false
	}
	_, ok = ifaceSet[ifacePkgPath]
	return ok
}

func importsPackage(pkg *types.Package, path string) bool {
	if pkg == nil || path == "" {
		return false
	}
	for _, imported := range pkg.Imports() {
		if imported != nil && imported.Path() == path {
			return true
		}
	}
	return false
}

func (a *analyzerState) checkAssertionsRequireAssertions(
	pass *analysis.Pass,
	ownership config.CompiledOwnershipConfig,
	assertionsCfg config.CompiledAssertionsConfig,
	exclude config.CompiledExcludeConfig,
	assertionInfos []assertionInfo,
) error {
	if ownership.ContractScope == nil {
		return nil
	}

	implTypes := filterImplementationTypesForRequireAssertions(pass, exclude)
	if len(implTypes) == 0 {
		return nil
	}

	candidateIfaces := collectReferencedInterfaces(pass, exclude)
	globalIfaces, globalCoImports, err := a.loadGlobalState(pass, ownership, exclude)
	if err != nil {
		return err
	}
	for _, ifaceInfo := range globalIfaces {
		ifacePkg := ifaceInfo.typeName.Pkg()
		if ifacePkg == nil {
			continue
		}
		if ifacePkg.Path() == pass.Pkg.Path() {
			continue
		}
		if _, exists := candidateIfaces[ifaceInfo.fullName]; exists {
			continue
		}
		candidateIfaces[ifaceInfo.fullName] = ifaceInfo
	}
	if len(candidateIfaces) == 0 {
		return nil
	}

	for _, ifaceInfo := range candidateIfaces {
		if a.shouldSkipRequireAssertionsInterface(ownership, ifaceInfo) {
			continue
		}
		ifacePkg := ifaceInfo.typeName.Pkg()
		if ifacePkg == nil {
			continue
		}
		if !assertionsCfg.RequireAssertionsStrict {
			if !interfaceIsRelevantForImpl(pass, ifacePkg.Path(), globalCoImports) {
				continue
			}
		}
		for _, implType := range implTypes {
			reportMissingAssertionIfNeeded(pass, ifaceInfo, implType, assertionInfos)
		}
	}

	return nil
}

func filterImplementationTypesForRequireAssertions(
	pass *analysis.Pass,
	exclude config.CompiledExcludeConfig,
) []*implTypeInfo {
	implTypes := collectImplementationTypes(pass)
	if len(implTypes) == 0 {
		return nil
	}

	filtered := make([]*implTypeInfo, 0, len(implTypes))
	seen := make(map[string]struct{}, len(implTypes))
	for _, implType := range implTypes {
		if isFileExcluded(pass.Fset, implType.pos, exclude) {
			continue
		}
		implObj := implType.named.Obj()
		if implObj == nil {
			continue
		}
		implPkg := implObj.Pkg()
		if implPkg == nil {
			continue
		}

		implFullName := implPkg.Path() + "." + implObj.Name()
		if isTypeExcluded(exclude, implFullName) {
			continue
		}
		if _, exists := seen[implFullName]; exists {
			continue
		}
		seen[implFullName] = struct{}{}

		filtered = append(filtered, implType)
	}

	return filtered
}

func (a *analyzerState) shouldSkipRequireAssertionsInterface(
	ownership config.CompiledOwnershipConfig,
	ifaceInfo *interfaceRefInfo,
) bool {
	if isOwnershipInterfaceIgnored(ownership, ifaceInfo.fullName, ifaceInfo.named) {
		return true
	}
	return !a.isContractualInterfaceForRequireAssertions(ownership, ifaceInfo)
}

func reportMissingAssertionIfNeeded(
	pass *analysis.Pass,
	ifaceInfo *interfaceRefInfo,
	implType *implTypeInfo,
	assertions []assertionInfo,
) {
	if !implementsInterfaceForRequireAssertions(implType.named, ifaceInfo.iface) {
		return
	}
	if hasSatisfyingAssertion(implType.named, ifaceInfo, assertions) {
		return
	}

	implObj := implType.named.Obj()
	ifaceObj := ifaceInfo.typeName
	if implObj.Pkg() == nil || ifaceObj.Pkg() == nil {
		return
	}

	msg := fmt.Sprintf(
		"%s: missing compile-time assertion for type %q implementing interface %q. "+
			"Use \"var _ %s.%s = (*%s)(nil)\" in package %q to assert",
		assertionsMissID,
		implObj.Name(),
		ifaceObj.Name(),
		ifaceObj.Pkg().Name(),
		ifaceObj.Name(),
		implObj.Name(),
		pass.Pkg.Name(),
	)
	reportDiagnostic(pass, implType.pos, assertionsMissID, msg)
}

func (a *analyzerState) isContractualInterfaceForRequireAssertions(
	ownership config.CompiledOwnershipConfig,
	ifaceInfo *interfaceRefInfo,
) bool {
	if ownership.ContractScope == nil {
		return false
	}

	switch *ownership.ContractScope {
	case config.ContractScopeAnyExported:
		return ifaceInfo.typeName.Exported()
	case config.ContractScopeExportedOutput:
		// Package-local analysis cannot observe external package exports; treat referenced
		// exported interfaces as contractual for requireassertions in this mode.
		return ifaceInfo.typeName.Exported()
	default:
		return false
	}
}

func hasSatisfyingAssertion(
	implType *types.Named,
	ifaceInfo *interfaceRefInfo,
	assertions []assertionInfo,
) bool {
	implObj := implType.Obj()
	for _, assertion := range assertions {
		if assertion.implType == nil || assertion.lhsType == nil || assertion.implType.Obj() != implObj {
			continue
		}
		if assertionSatisfiesInterface(assertion.lhsType, ifaceInfo) {
			return true
		}
	}
	return false
}

func assertionSatisfiesInterface(lhsType types.Type, ifaceInfo *interfaceRefInfo) bool {
	switch t := lhsType.(type) {
	case *types.Named:
		if _, ok := t.Underlying().(*types.Interface); !ok {
			return false
		}
		owner := typeutil.UnaliasTypeName(t.Obj())
		if owner == nil {
			return false
		}
		ownerPkg := owner.Pkg()
		if ownerPkg == nil {
			return false
		}
		return ownerPkg.Path()+"."+owner.Name() == ifaceInfo.fullName
	case *types.Interface:
		return interfacesEquivalent(t, ifaceInfo.iface)
	default:
		return false
	}
}

func implementsInterfaceForRequireAssertions(implType *types.Named, iface *types.Interface) bool {
	if implType == nil || iface == nil {
		return false
	}
	implObj := implType.Obj()
	if implObj == nil || implObj.Pkg() == nil {
		return false
	}
	ifaceMethods, ok := interfaceMethodSignaturesForImpl(iface, implObj.Pkg().Path())
	if !ok {
		return false
	}

	if methodSetSatisfiesInterface(types.NewMethodSet(implType), ifaceMethods) {
		return true
	}
	return methodSetSatisfiesInterface(types.NewMethodSet(types.NewPointer(implType)), ifaceMethods)
}

func interfaceMethodSignaturesForImpl(
	iface *types.Interface,
	implPkgPath string,
) (map[string]string, bool) {
	if iface == nil {
		return nil, false
	}
	iface = iface.Complete()
	result := make(map[string]string, iface.NumMethods())
	for i := range iface.NumMethods() {
		method := iface.Method(i)
		if method == nil {
			continue
		}
		if !method.Exported() {
			pkg := method.Pkg()
			if pkg == nil || pkg.Path() != implPkgPath {
				return nil, false
			}
		}
		sig, ok := method.Type().(*types.Signature)
		if !ok {
			continue
		}
		result[methodKey(method)] = signatureString(stripReceiver(sig))
	}
	return result, true
}

func methodSetSatisfiesInterface(
	methodSet *types.MethodSet,
	ifaceMethods map[string]string,
) bool {
	if methodSet == nil {
		return false
	}
	implMethods := methodSetSignatures(methodSet)
	if len(implMethods) < len(ifaceMethods) {
		return false
	}
	for key, ifaceSig := range ifaceMethods {
		implSig, ok := implMethods[key]
		if !ok || implSig != ifaceSig {
			return false
		}
	}
	return true
}

func methodSetSignatures(methodSet *types.MethodSet) map[string]string {
	result := make(map[string]string, methodSet.Len())
	for i := range methodSet.Len() {
		selection := methodSet.At(i)
		fn, ok := selection.Obj().(*types.Func)
		if !ok {
			continue
		}
		sig, ok := fn.Type().(*types.Signature)
		if !ok {
			continue
		}
		key := methodKey(fn)
		if key == "" {
			continue
		}
		result[key] = signatureString(stripReceiver(sig))
	}
	return result
}

func interfacesEquivalent(left *types.Interface, right *types.Interface) bool {
	if left == nil || right == nil {
		return false
	}
	leftMap := interfaceSignatureMap(left)
	rightMap := interfaceSignatureMap(right)
	if len(leftMap) != len(rightMap) {
		return false
	}
	for key, leftSig := range leftMap {
		if rightSig, ok := rightMap[key]; !ok || rightSig != leftSig {
			return false
		}
	}
	return true
}

func interfaceSignatureMap(iface *types.Interface) map[string]string {
	if iface == nil {
		return nil
	}
	iface = iface.Complete()
	result := make(map[string]string, iface.NumMethods())
	for i := range iface.NumMethods() {
		method := iface.Method(i)
		if method == nil {
			continue
		}
		sig, ok := method.Type().(*types.Signature)
		if !ok {
			continue
		}
		result[methodKey(method)] = signatureString(stripReceiver(sig))
	}
	return result
}

func methodKey(fn *types.Func) string {
	if fn == nil {
		return ""
	}
	if fn.Exported() {
		return fn.Name()
	}
	pkg := fn.Pkg()
	if pkg == nil {
		return ""
	}
	return pkg.Path() + "." + fn.Name()
}

func signatureString(sig *types.Signature) string {
	if sig == nil {
		return ""
	}
	return types.TypeString(sig, func(pkg *types.Package) string {
		if pkg == nil {
			return ""
		}
		return pkg.Path()
	})
}

func stripReceiver(sig *types.Signature) *types.Signature {
	if sig == nil || sig.Recv() == nil {
		return sig
	}
	return types.NewSignatureType(
		nil,
		nil,
		typeParamListToSlice(sig.TypeParams()),
		sig.Params(),
		sig.Results(),
		sig.Variadic(),
	)
}

func typeParamListToSlice(list *types.TypeParamList) []*types.TypeParam {
	if list == nil || list.Len() == 0 {
		return nil
	}
	result := make([]*types.TypeParam, list.Len())
	for i := range list.Len() {
		result[i] = list.At(i)
	}
	return result
}

func isOwnershipInterfaceIgnored(cfg config.CompiledOwnershipConfig, fullName string, named *types.Named) bool {
	if matchesAnyPattern(fullName, cfg.IgnoreInterfaces) {
		return true
	}
	return cfg.IgnoreMarkerInterfaces && typeutil.IsMarkerInterface(named)
}

func collectPackageTypes(
	pass *analysis.Pass,
	exclude config.CompiledExcludeConfig,
) ([]*interfaceInfo, []*types.Named) {
	var interfaces []*interfaceInfo
	var implCandidates []*types.Named
	for _, file := range pass.Files {
		if isFileExcluded(pass.Fset, file.Pos(), exclude) {
			continue
		}
		collectTypesFromFile(pass, file, &interfaces, &implCandidates)
	}
	return interfaces, implCandidates
}

func collectImplementationTypes(pass *analysis.Pass) []*implTypeInfo {
	var implTypes []*implTypeInfo
	for _, file := range pass.Files {
		collectImplementationTypesFromFile(pass, file, &implTypes)
	}
	return implTypes
}

func collectImplementationTypesFromFile(
	pass *analysis.Pass,
	file *ast.File,
	implTypes *[]*implTypeInfo,
) {
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			obj := pass.TypesInfo.Defs[typeSpec.Name]
			typeName, ok := obj.(*types.TypeName)
			if !ok {
				continue
			}
			if typeName.IsAlias() {
				continue
			}

			named, ok := typeName.Type().(*types.Named)
			if !ok {
				continue
			}
			if _, ok := named.Underlying().(*types.Interface); ok {
				continue
			}

			*implTypes = append(*implTypes, &implTypeInfo{
				named: named,
				pos:   typeSpec.Name.Pos(),
			})
		}
	}
}

func collectReferencedInterfaces(
	pass *analysis.Pass,
	exclude config.CompiledExcludeConfig,
) map[string]*interfaceRefInfo {
	result := make(map[string]*interfaceRefInfo)
	if pass.TypesInfo == nil {
		return result
	}

	for ident, obj := range pass.TypesInfo.Uses {
		typeName, ok := obj.(*types.TypeName)
		if !ok {
			continue
		}
		if isFileExcluded(pass.Fset, ident.Pos(), exclude) {
			continue
		}

		ownerTypeName := typeutil.UnaliasTypeName(typeName)
		if ownerTypeName == nil {
			continue
		}

		ownerPkg := ownerTypeName.Pkg()
		if ownerPkg == nil {
			continue
		}
		if ownerPkg.Path() == pass.Pkg.Path() {
			continue
		}

		named, ok := ownerTypeName.Type().(*types.Named)
		if !ok {
			continue
		}
		iface, ok := named.Underlying().(*types.Interface)
		if !ok {
			continue
		}

		fullName := ownerPkg.Path() + "." + ownerTypeName.Name()
		if isTypeExcluded(exclude, fullName) {
			continue
		}
		if _, exists := result[fullName]; exists {
			continue
		}

		result[fullName] = &interfaceRefInfo{
			typeName: ownerTypeName,
			named:    named,
			iface:    iface,
			fullName: fullName,
		}
	}

	return result
}

func reportDiagnostic(
	pass *analysis.Pass,
	pos token.Pos,
	category string,
	message string,
) {
	//nolint:exhaustruct // optional fields use zero values
	pass.Report(analysis.Diagnostic{
		Pos:      pos,
		Category: category,
		Message:  message,
	})
}

func reportOwnershipViolation(
	pass *analysis.Pass,
	pos token.Pos,
	ifaceName string,
	implTypeName string,
) {
	msg := fmt.Sprintf(
		"%s: interface %q has implementation %q in same package; "+
			"move interface to consumer package or dedicated contract package",
		ownershipID,
		ifaceName,
		implTypeName,
	)
	reportDiagnostic(pass, pos, ownershipID, msg)
}

// checkOwnershipExportedOutput scans the package for interface ownership violations
// using the exportedoutput mode: an interface is contractual if it appears in
// exported outputs (function results, method results, variable types).
func (a *analyzerState) checkOwnershipExportedOutput(
	pass *analysis.Pass,
	cfg config.CompiledOwnershipConfig,
	exclude config.CompiledExcludeConfig,
) {
	pkgPath := pass.Pkg.Path()

	// Skip packages designated as contract packages.
	if matchesAnyPattern(pkgPath, cfg.ContractPackages) {
		return
	}

	// Collect all interfaces and implementation candidates from this package.
	interfaces, implCandidates := collectPackageTypes(pass, exclude)

	if len(interfaces) == 0 || len(implCandidates) == 0 {
		return
	}

	// Collect exported output surfaces for each interface.
	// Key: interface TypeName, Value: appears in output (true) or only in input (false).
	interfaceInOutput := make(map[*types.TypeName]bool)

	for _, file := range pass.Files {
		if isFileExcluded(pass.Fset, file.Pos(), exclude) {
			continue
		}
		collectExportedOutputSurfaces(pass, file, interfaces, interfaceInOutput, cfg.SkipIfUsedAsInput)
	}

	// For each interface, check if it's contractual and has a local implementation.
	for _, ifaceInfo := range interfaces {
		// Skip unexported interfaces.
		if !ifaceInfo.typeName.Exported() {
			continue
		}

		fullName := pkgPath + "." + ifaceInfo.typeName.Name()

		if isTypeExcluded(exclude, fullName) {
			continue
		}

		// Skip interfaces excluded by ignoreinterfaces/ignoremarkerinterfaces.
		if isOwnershipInterfaceIgnored(cfg, fullName, ifaceInfo.named) {
			continue
		}

		// Check if interface appears in exported outputs.
		appearsInOutput, found := interfaceInOutput[ifaceInfo.typeName]
		if !found || !appearsInOutput {
			continue
		}

		// Check if any local type implements this interface.
		implementer := findFirstLocalImplementer(ifaceInfo.iface, implCandidates, pkgPath, exclude)
		if implementer == nil {
			continue
		}

		// Report violation.
		reportOwnershipViolation(pass, ifaceInfo.pos, ifaceInfo.typeName.Name(), implementer.Obj().Name())
	}
}

// checkOwnershipAnyExported scans the package for interface ownership violations
// using the anyexported mode: any exported interface is considered contractual.
func (a *analyzerState) checkOwnershipAnyExported(
	pass *analysis.Pass,
	cfg config.CompiledOwnershipConfig,
	exclude config.CompiledExcludeConfig,
) {
	pkgPath := pass.Pkg.Path()

	// Skip packages designated as contract packages.
	if matchesAnyPattern(pkgPath, cfg.ContractPackages) {
		return
	}

	// Collect all interfaces and implementation candidates from this package.
	interfaces, implCandidates := collectPackageTypes(pass, exclude)

	if len(interfaces) == 0 || len(implCandidates) == 0 {
		return
	}

	// For each exported interface, check if it has a local implementation.
	for _, ifaceInfo := range interfaces {
		// Skip unexported interfaces.
		if !ifaceInfo.typeName.Exported() {
			continue
		}

		fullName := pkgPath + "." + ifaceInfo.typeName.Name()

		if isTypeExcluded(exclude, fullName) {
			continue
		}

		// Skip interfaces excluded by ignoreinterfaces/ignoremarkerinterfaces.
		if isOwnershipInterfaceIgnored(cfg, fullName, ifaceInfo.named) {
			continue
		}

		// Check if any local type implements this interface.
		implementer := findFirstLocalImplementer(ifaceInfo.iface, implCandidates, pkgPath, exclude)
		if implementer == nil {
			continue
		}

		// Report violation.
		reportOwnershipViolation(pass, ifaceInfo.pos, ifaceInfo.typeName.Name(), implementer.Obj().Name())
	}
}

// collectExportedOutputSurfaces scans a file and marks interfaces that appear in:
// - Exported variable types
// - Exported function results
// - Exported method results
// When skipInput is true, interfaces that appear ONLY in input params are not marked.
func collectExportedOutputSurfaces(
	pass *analysis.Pass,
	file *ast.File,
	interfaces []*interfaceInfo,
	result map[*types.TypeName]bool,
	skipInput bool,
) {
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			collectFromGenDecl(pass, d, interfaces, result, skipInput)
		case *ast.FuncDecl:
			collectFromFuncDecl(pass, d, interfaces, result, skipInput)
		}
	}
}

func collectFromGenDecl(
	pass *analysis.Pass,
	decl *ast.GenDecl,
	interfaces []*interfaceInfo,
	result map[*types.TypeName]bool,
	skipInput bool,
) {
	if decl.Tok == token.VAR {
		collectFromVarDecl(pass, decl, interfaces, result)
	}
	if decl.Tok == token.TYPE {
		collectFromTypeDecl(pass, decl, interfaces, result, skipInput)
	}
}

func collectFromVarDecl(
	pass *analysis.Pass,
	decl *ast.GenDecl,
	interfaces []*interfaceInfo,
	result map[*types.TypeName]bool,
) {
	for _, spec := range decl.Specs {
		valueSpec, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		for _, name := range valueSpec.Names {
			if !name.IsExported() {
				continue
			}

			obj := pass.TypesInfo.Defs[name]
			if obj == nil {
				continue
			}
			markInterfacesInType(obj.Type(), interfaces, result)
		}
	}
}

func collectFromTypeDecl(
	pass *analysis.Pass,
	decl *ast.GenDecl,
	interfaces []*interfaceInfo,
	result map[*types.TypeName]bool,
	skipInput bool,
) {
	for _, spec := range decl.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok || !typeSpec.Name.IsExported() {
			continue
		}

		obj := pass.TypesInfo.Defs[typeSpec.Name]
		typeName, ok := obj.(*types.TypeName)
		if !ok {
			continue
		}

		named, ok := typeName.Type().(*types.Named)
		if !ok {
			continue
		}

		for i := range named.NumMethods() {
			method := named.Method(i)
			if !method.Exported() {
				continue
			}
			sig, ok := method.Type().(*types.Signature)
			if !ok {
				continue
			}
			markInterfacesInSignature(sig, interfaces, result, skipInput)
		}
	}
}

// collectFromFuncDecl handles package-level function declarations.
// Methods are handled separately via collectFromTypeDecl.
func collectFromFuncDecl(
	pass *analysis.Pass,
	decl *ast.FuncDecl,
	interfaces []*interfaceInfo,
	result map[*types.TypeName]bool,
	skipInput bool,
) {
	if !decl.Name.IsExported() || decl.Recv != nil {
		return
	}

	obj := pass.TypesInfo.Defs[decl.Name]
	if obj == nil {
		return
	}
	fn, ok := obj.(*types.Func)
	if !ok {
		return
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return
	}
	markInterfacesInSignature(sig, interfaces, result, skipInput)
}

// markInterfacesInSignature marks interfaces that appear in a function signature.
// When skipInput is true, interfaces that appear ONLY in params (not results) get a false entry.
// When skipInput is false, interfaces in params are also marked as true.
func markInterfacesInSignature(
	sig *types.Signature,
	interfaces []*interfaceInfo,
	result map[*types.TypeName]bool,
	skipInput bool,
) {
	inResults := collectInterfacesInTuple(sig.Results(), interfaces)
	inParams := collectInterfacesInTuple(sig.Params(), interfaces)

	for _, ifaceInfo := range interfaces {
		typeName := ifaceInfo.typeName
		if inResults[typeName] {
			result[typeName] = true
			continue
		}
		if !inParams[typeName] {
			continue
		}

		if skipInput {
			if _, found := result[typeName]; !found {
				result[typeName] = false
			}
			continue
		}

		result[typeName] = true
	}
}

// collectInterfacesInTuple returns a set of interfaces found in a tuple.
func collectInterfacesInTuple(
	tuple *types.Tuple,
	interfaces []*interfaceInfo,
) map[*types.TypeName]bool {
	found := make(map[*types.TypeName]bool)
	if tuple == nil {
		return found
	}
	for i := range tuple.Len() {
		markInterfacesInType(tuple.At(i).Type(), interfaces, found)
	}
	return found
}

func markInterfacesInType(
	typ types.Type,
	interfaces []*interfaceInfo,
	result map[*types.TypeName]bool,
) {
	if typ == nil {
		return
	}
	for _, ifaceInfo := range interfaces {
		if typeutil.ContainsInterface(ifaceInfo.typeName, typ) {
			result[ifaceInfo.typeName] = true
		}
	}
}

// interfaceInfo holds information about an interface declaration.
type interfaceInfo struct {
	typeName *types.TypeName
	named    *types.Named
	iface    *types.Interface
	pos      token.Pos
}

type implTypeInfo struct {
	named *types.Named
	pos   token.Pos
}

type interfaceRefInfo struct {
	typeName *types.TypeName
	named    *types.Named
	iface    *types.Interface
	fullName string
}

type assertionInfo struct {
	implType *types.Named
	lhsType  types.Type
}

// collectTypesFromFile extracts interfaces and implementation candidates from a file.
func collectTypesFromFile(
	pass *analysis.Pass,
	file *ast.File,
	interfaces *[]*interfaceInfo,
	implCandidates *[]*types.Named,
) {
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			obj := pass.TypesInfo.Defs[typeSpec.Name]
			typeName, ok := obj.(*types.TypeName)
			if !ok {
				continue
			}
			if typeName.IsAlias() {
				// Skip type aliases - ownership belongs to original.
				continue
			}

			named, ok := typeName.Type().(*types.Named)
			if !ok {
				continue
			}

			// Check if this is an interface or a potential implementation.
			iface, ok := named.Underlying().(*types.Interface)
			if ok {
				*interfaces = append(*interfaces, &interfaceInfo{
					typeName: typeName,
					named:    named,
					iface:    iface,
					pos:      typeSpec.Name.Pos(),
				})
				continue
			}

			// Non-interface named type - potential implementation.
			*implCandidates = append(*implCandidates, named)
		}
	}
}

// findFirstLocalImplementer returns the first candidate that implements the interface.
// Returns nil if no implementers found.
func findFirstLocalImplementer(
	iface *types.Interface,
	candidates []*types.Named,
	pkgPath string,
	exclude config.CompiledExcludeConfig,
) *types.Named {
	for _, candidate := range candidates {
		candidateObj := candidate.Obj()
		if candidateObj == nil {
			continue
		}

		fullName := pkgPath + "." + candidateObj.Name()
		if isTypeExcluded(exclude, fullName) {
			continue
		}

		if typeutil.ImplementsEither(candidate, iface) {
			return candidate
		}
	}
	return nil
}
