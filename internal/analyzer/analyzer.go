// Package analyzer provides the ifaceguard analysis.Analyzer for detecting
// interface architecture violations in Go code.
package analyzer

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"regexp"

	"github.com/n-r-w/ifaceguard/internal/config"
	"github.com/n-r-w/ifaceguard/internal/typeutil"
	"golang.org/x/tools/go/analysis"
)

// AnalyzerName is the identifier used for the analyzer in reports and configuration.
const (
	AnalyzerName     = "ifaceguard"
	ownershipID      = "IFG001-OWNERSHIP"
	assertionsID     = "IFG002-ASSERTION-PLACEMENT"
	assertionsMissID = "IFG003-ASSERTION-MISSING"
)

// analyzerState holds configuration and dependencies for the analyzer.
type analyzerState struct {
	cfg config.Config
}

// New creates a new ifaceguard analyzer.
// The cfg parameter provides rule configuration; use config.Default() for defaults.
func New(cfg config.Config) (*analysis.Analyzer, error) {
	return newAnalyzer(cfg)
}

func newAnalyzer(cfg config.Config) (*analysis.Analyzer, error) {
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
		a.checkAssertions(pass, compiled.Ownership, compiled.Assertions, compiled.Exclude)
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
) {
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
		a.checkAssertionsRequireAssertions(pass, ownership, exclude, assertions)
	}
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

		infos = append(infos, assertionInfo{
			implType: implType,
			lhsType:  lhsType,
		})

		// B3: current package must equal implementation package.
		currentPkg := pass.Pkg
		if implPkg.Path() == currentPkg.Path() {
			continue // correct placement, no violation
		}

		// B4: check wiringpackages exclusion.
		if matchesAnyPattern(currentPkg.Path(), cfg.WiringPackages) {
			continue
		}

		// Report violation at the variable declaration position.
		msg := fmt.Sprintf(
			"%s: compile-time assertion for type %s.%s should be in package %s "+
				"(or in an allowed wiring package). Current package: %s.",
			assertionsID,
			implPkg.Name(), implType.Obj().Name(),
			implPkg.Name(),
			currentPkg.Name(),
		)
		reportDiagnostic(pass, name.Pos(), assertionsID, msg)
	}

	return infos
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

func (a *analyzerState) checkAssertionsRequireAssertions(
	pass *analysis.Pass,
	ownership config.CompiledOwnershipConfig,
	exclude config.CompiledExcludeConfig,
	assertionInfos []assertionInfo,
) {
	if ownership.ContractScope == nil {
		return
	}

	implTypes := filterImplementationTypesForRequireAssertions(pass, exclude)
	if len(implTypes) == 0 {
		return
	}

	candidateIfaces := collectReferencedInterfaces(pass, exclude)
	if len(candidateIfaces) == 0 {
		return
	}

	for _, ifaceInfo := range candidateIfaces {
		if a.shouldSkipRequireAssertionsInterface(ownership, ifaceInfo) {
			continue
		}
		for _, implType := range implTypes {
			reportMissingAssertionIfNeeded(pass, ifaceInfo, implType, assertionInfos)
		}
	}
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
	if !typeutil.ImplementsEither(implType.named, ifaceInfo.iface) {
		return
	}
	if hasSatisfyingAssertion(implType.named, ifaceInfo, assertions) {
		return
	}

	implPkg := implType.named.Obj().Pkg()
	ifacePkg := ifaceInfo.typeName.Pkg()
	if implPkg == nil || ifacePkg == nil {
		return
	}

	msg := fmt.Sprintf(
		"%s: missing compile-time assertion for type %s.%s implementing interface %s.%s.",
		assertionsMissID,
		implPkg.Path(), implType.named.Obj().Name(),
		ifacePkg.Path(), ifaceInfo.typeName.Name(),
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
		return owner == ifaceInfo.typeName
	case *types.Interface:
		return types.Identical(t, ifaceInfo.iface)
	default:
		return false
	}
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
	fullName string,
	pkgPath string,
	implTypeName string,
) {
	msg := fmt.Sprintf(
		"%s: interface %s has implementation %s.%s in same package; "+
			"move interface to consumer package or dedicated contract package",
		ownershipID,
		fullName,
		pkgPath, implTypeName,
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
		reportOwnershipViolation(pass, ifaceInfo.pos, fullName, pkgPath, implementer.Obj().Name())
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
		reportOwnershipViolation(pass, ifaceInfo.pos, fullName, pkgPath, implementer.Obj().Name())
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
