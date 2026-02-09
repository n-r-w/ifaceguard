# ifaceguard

A Go linter that checks architectural properties of interface usage.

## Purpose

`ifaceguard` enforces two key architectural principles:

- **Dependency Inversion Principle (DIP):** Interface contracts should belong to the consumer side (or a dedicated contract package), not the implementation side.
- **Explicit compile-time assertions:** Implementation packages must declare compile-time assertions verifying interface compliance.

## How the two checks differ

`ifaceguard` runs two independent checks that target different problems:

- **Ownership check (IFG001-OWNERSHIP):** answers the question “*Is the interface declared in the right package?*”.
  It reports when a **contractual** interface (as defined by `ownership.contractscope`) is declared in the same package that also declares a type implementing it, unless the package is explicitly allowed (for example, `ownership.contractpackages`).

- **Assertions check (IFG002-ASSERTION-PLACEMENT / IFG003-ASSERTION-MISSING):** answers the question “*Is the compile-time assertion placed (or present) in the right package?*”.
  It reports an assertion placed outside the implementation package, and optionally reports a missing assertion when `assertions.requireassertions=true`.

These checks can be enabled or disabled independently via `ownership.enabled` and `assertions.enabled`.

## Project Status

`ifaceguard` is in early development. The core analysis logic is implemented and tested, but the configuration and CLI are still evolving. Breaking changes to config keys and behavior are expected until the first stable release.

## Installation

```bash
go install github.com/n-r-w/ifaceguard/cmd/ifaceguard@latest
```

### From releases

Prebuilt artifacts are published on the [GitHub Releases page](https://github.com/n-r-w/ifaceguard/releases).

Release assets include:

- **Standalone `ifaceguard` binary**, named `ifaceguard_<version>_<os>_<arch>`.
- **Custom golangci-lint binary** built with the plugin, named `custom-gcl_<os>_<arch>` (Windows adds `.exe`).

Choose the asset that matches your OS/arch, extract it if needed, and place the binary on your `PATH`.
On macOS, if the system blocks the binary, open System Settings > Privacy & Security and allow it, then run again.

## Usage

### Standalone

```bash
ifaceguard [flags] [packages]
```

Standalone runs accept an optional config file:

```bash
ifaceguard -config /path/to/ifaceguard.yml [packages]
```

If no packages are provided, the default is `./...`.

If `-config` is omitted, ifaceguard uses the default settings.

The config file uses the same settings keys as the golangci-lint example below, but at the top level:

```yaml
ownership:
  enabled: true
  contractscope: exportedoutput
  skipifusedasinput: true
  contractpackages: []
  ignoreinterfaces: []
  ignoremarkerinterfaces: true
assertions:
  enabled: true
  wiringpackages: []
  acceptconversiononlyform: false
  scanfunctionbodies: false
  requireassertions: false
  requireassertionsstrict: false
exclude:
  # Regexes applied to full file paths
  files:
    - ".*_mock\\.go$"
  # Regexes applied to full type names: pkgpath.TypeName
  types:
    - ".*\\.Mock.*$"
```


### golangci-lint (module plugin)

For **local development** (plugin not integrated into official golangci-lint):

Create `.custom-gcl.yml` in your project root:

```yaml
plugins:
  - module: 'github.com/n-r-w/ifaceguard'
    import: 'github.com/n-r-w/ifaceguard/pkg/golangci'
    path: /absolute/path/to/ifaceguard  # Use absolute path
```

Note: in this repository, use `task build` to build both the standalone binary and the custom golangci-lint binary. The Taskfile uses `.golangci-version` and the repo's `build/.custom-gcl.yml`.

For **published plugin** (after release):

```yaml
plugins:
  - module: 'github.com/n-r-w/ifaceguard'
    import: 'github.com/n-r-w/ifaceguard/pkg/golangci'
    version: latest
```

Build the custom golangci-lint binary:

```bash
task build
```

Then enable in your `.golangci.yml`:

```yaml
linters:
  enable:
    - ifaceguard

settings:
  custom:
    ifaceguard:
      type: module

      # Example configuration
      settings:
        ownership:
          enabled: true
          contractscope: exportedoutput
          skipifusedasinput: true
          contractpackages:
            - "^example\\.com/project/(contract|port)(/|$)"
          ignoreinterfaces:
            - ".*\\.Error$"
          ignoremarkerinterfaces: true
        assertions:
          enabled: true
          wiringpackages:
            - "^example\\.com/project/(cmd|internal/compose)(/|$)"
          acceptconversiononlyform: false
          scanfunctionbodies: false
          requireassertions: false
          requireassertionsstrict: false
        exclude:
          # Regexes applied to full file paths
          files:
            - ".*_mock\\.go$"
          # Regexes applied to full type names: pkgpath.TypeName
          types:
            - ".*\\.Mock.*$"
```

### Configuration options

#### `ownership`

- `enabled` (bool, default: true): toggles the ownership rule (IFG001). When false, no ownership diagnostics are reported.
- `contractscope` (enum: `exportedoutput|anyexported`, default: `exportedoutput`): defines which interfaces are treated as **contractual** for the ownership rule.
  - `exportedoutput`: an interface is contractual only if it appears in the package’s exported API outputs (exported function/method results or exported variables). By default, interfaces used **only** in exported inputs are ignored unless `skipifusedasinput=false`.
  - `anyexported`: every exported interface is contractual, regardless of where it is used.
- `skipifusedasinput` (bool, default: true): only applies to `contractscope=exportedoutput`. If true, interfaces that appear only in exported inputs (parameters) are **not** considered contractual. Rationale: input-only interfaces typically describe dependencies the package consumes, so this avoids ownership violations for consumer-side contracts. Set to false to treat exported inputs as part of the public contract (stricter DIP). Note: if the interface appears in exported outputs (including nested inside returned/variable types), it is **not** input-only and this flag does not suppress diagnostics.
- `contractpackages` (list[regex], default: empty): regexes matched against interface package import paths. Matching packages are treated as allowed contract packages, so ownership violations are not reported there.
- `ignoreinterfaces` (list[regex], default: empty): regexes matched against full interface names (`pkgpath.Interface`) to exclude them from the ownership rule.
- `ignoremarkerinterfaces` (bool, default: true): if true, marker interfaces (no methods and no embedded interfaces, and **not** a type-set constraint) are excluded from the ownership rule. Marker interfaces are named empty interfaces used for readability; any type is assignable to them, so they do not enforce behavior. If you need enforcement, add a method or a type-set constraint instead.

#### `assertions`

- `enabled` (bool, default: true): toggles assertion checks (IFG002) and missing-assertion checks (IFG003). When false, no assertion diagnostics are reported.
- `wiringpackages` (list[regex], default: empty): regexes matched against package import paths where assertions are allowed outside the implementation package (e.g., wiring/compose packages).
- `acceptconversiononlyform` (bool, default: false): if true, also treats `var _ = iface((*T)(nil))` as a valid assertion form (conversion-only form).
- `scanfunctionbodies` (bool, default: false): if true, scans inside function bodies for `var _ I = ...` assertions. When false, only package-level `var` declarations are scanned.
- `requireassertions` (bool, default: false): if true, requires at least one recognized assertion in the **implementation package** for each type that implements a contractual interface from another package; reports IFG003 when missing.
  When enabled, ifaceguard scans all packages in the current module to find contractual interfaces, so missing assertions are reported even if the implementation package does not reference the interface directly.
- `requireassertionsstrict` (bool, default: false): if true, disables the relevance filter for IFG003 and reports missing assertions for any matching contractual interface in the module, even when no direct import/co-import evidence exists.

#### `exclude`

- `files` (list[regex], default: empty): regexes matched against full file paths (with `/` separators). Matching files are skipped entirely.
- `types` (list[regex], default: empty): regexes matched against full type names (`pkgpath.TypeName`). Matching types are ignored by both rules.

Notes:
- Test files are analyzed by default; use `exclude.files` to omit them if needed.
- Unknown configuration keys are rejected with an error.
- Assertion placement checks scan any `var` declarations (including grouped `var (...)`); `wiringpackages` depends only on the package path, not on the declaration form.

### Examples: how settings affect diagnostics

The examples below use these diagnostic IDs:
- `IFG001-OWNERSHIP`: interface ownership violation.
- `IFG002-ASSERTION-PLACEMENT`: assertion placed outside the implementation package.
- `IFG003-ASSERTION-MISSING`: missing assertion when `requireassertions=true`.

#### Ownership: `contractscope` and `skipifusedasinput`

Code:

```go
// provider/provider.go
package provider

type Runner interface {
	Run() error
}

type Service struct{}

func (Service) Run() error { return nil }

// Exported API uses Runner only as input.
func Use(r Runner) {}
```

Config and expected diagnostics:

```yaml
ownership:
  contractscope: exportedoutput
  skipifusedasinput: true
```

- Expected: **no** `IFG001-OWNERSHIP` (input-only usage is ignored).

```yaml
ownership:
  contractscope: exportedoutput
  skipifusedasinput: false
```

- Expected: `IFG001-OWNERSHIP` on `provider.Runner`.

```yaml
ownership:
  contractscope: anyexported
```

- Expected: `IFG001-OWNERSHIP` on `provider.Runner`.

With `exportedoutput`, ifaceguard treats an interface as contractual only if it appears in exported results or exported variables. If it shows up only in exported inputs, it is ignored unless you set `skipifusedasinput=false`. With `anyexported`, any exported interface is contractual regardless of where it is used. This matches the idea that the public contract is primarily what the package returns/exports, while input-only dependencies are often consumer-side and should not force ownership unless you opt into stricter enforcement.

#### Ownership: `contractpackages`

Code:

```go
// contract/runner.go
package contract

type Runner interface {
  Run() error
}

type Service struct{}

func (Service) Run() error { return nil }

// NewRunner returns Runner, making it contractual in this package.
func NewRunner() Runner {
  return Service{}
}
```

Config and expected diagnostics:

```yaml
ownership:
  contractpackages:
    - "^example\\.com/project/contract(/|$)"
```

- Expected: **no** `IFG001-OWNERSHIP` (contract package is allowed).

Without `contractpackages`, the same code would report `IFG001-OWNERSHIP` on `contract.Runner` because the interface and its implementation live in the same package and the interface appears in exported output.

If the interface’s package path matches `contractpackages`, ifaceguard treats it as a designated contract package and does not report ownership violations for interfaces declared there, even when implementations are in the same package. This lets you centralize contracts (and occasional helper implementations) without violating the ownership rule.

#### Ownership: `ignoreinterfaces` and `ignoremarkerinterfaces`

Code:

```go
// provider/ignore.go
package provider

type Ignored interface {
	Run() error
}

type Service struct{}

func (Service) Run() error { return nil }
```

Config and expected diagnostics:

```yaml
ownership:
  ignoreinterfaces:
    - "example\\.com/project/provider\\.Ignored$"
```

- Expected: **no** `IFG001-OWNERSHIP` for `provider.Ignored`.

If `ignoremarkerinterfaces=true`, a marker interface (no methods and no embedded interfaces) is excluded from ownership checks in the same way.

Example of a marker interface case: if a package declares `type Marker interface{}` and exposes it via an exported API, IFG001 would be reported when `ignoremarkerinterfaces=false`. With `ignoremarkerinterfaces=true`, that marker interface is skipped.

Interfaces matching `ignoreinterfaces` are skipped entirely. When `ignoremarkerinterfaces=true`, marker interfaces (no methods/embeds and not type sets) are also skipped, so ownership diagnostics are not emitted for them. This is useful for explicit exceptions and to avoid noise from tag-like interfaces that do not enforce behavior.

#### Assertions: `wiringpackages`

Code:

```go
// internal/compose/assertions.go
package compose

import (
	"example.com/project/contract"
	"example.com/project/provider"
)

var _ contract.Runner = (*provider.Service)(nil)
```

Config and expected diagnostics:

```yaml
assertions:
  wiringpackages:
    - "^example\\.com/project/internal/compose(/|$)"
```

- Expected: **no** `IFG002-ASSERTION-PLACEMENT` for this assertion.

Without the `wiringpackages` match, the same code reports `IFG002-ASSERTION-PLACEMENT` at the `var _ ...` line.

Assertions are expected to live in the implementation package; `wiringpackages` allows specific compose/wiring packages to host assertions without IFG002. This reflects architectures where wiring/assembly is centralized and assertions are placed alongside wiring code.

#### Assertions: `requireassertions`

Code:

```go
// contract/runner.go
package contract

type Runner interface {
	Run() error
}
```

```go
// provider/service.go
package provider

import "example.com/project/contract"

type Service struct{}

func (Service) Run() error { return nil }

// No compile-time assertion here.
```

Config and expected diagnostics:

```yaml
assertions:
  requireassertions: true
```

- Expected: `IFG003-ASSERTION-MISSING` on `provider.Service`.

If `requireassertions=false`, no `IFG003-ASSERTION-MISSING` is reported.

When `requireassertions=true`, ifaceguard reports IFG003 unless each implementation of a contractual external interface has at least one compile-time assertion in the implementation package. The goal is to make conformance explicit and prevent silent drift.

#### Assertions: `acceptconversiononlyform`

Code:

```go
// contract/assert.go
package contract

import "example.com/project/provider"

type Runner interface {
	Run() error
}

var _ = Runner((*provider.Service)(nil))
```

Config and expected diagnostics:

```yaml
assertions:
  acceptconversiononlyform: true
```

- Expected: `IFG002-ASSERTION-PLACEMENT` at the `var _ = ...` line.

If `acceptconversiononlyform=false`, this assertion form is ignored and no `IFG002-ASSERTION-PLACEMENT` is reported.

When `acceptconversiononlyform=true`, the conversion-only assertion form counts as an assertion and is checked for placement; when false, that form is ignored. This lets teams choose whether to accept the more implicit conversion-only style.

#### Assertions: `scanfunctionbodies`

Code:

```go
// contract/assert.go
package contract

import "example.com/project/provider"

type Runner interface {
	Run() error
}

func init() {
	var _ Runner = (*provider.Service)(nil)
}
```

Config and expected diagnostics:

```yaml
assertions:
  scanfunctionbodies: true
```

- Expected: `IFG002-ASSERTION-PLACEMENT` for the `var _ ...` inside `init`.

If `scanfunctionbodies=false`, that in-function assertion is not scanned and no `IFG002-ASSERTION-PLACEMENT` is reported.

When `scanfunctionbodies=true`, ifaceguard scans inside function bodies for `var _ I = ...` assertions; when false, only package-level `var` declarations are scanned. This keeps the default focused on visible, package-level assertions while allowing deeper scans when needed.

#### Exclusions: `exclude.files` and `exclude.types`

Code:

```go
// provider/generated.go
package provider

type Runner interface {
	Run() error
}

type Service struct{}

func (Service) Run() error { return nil }
```

Config and expected diagnostics:

```yaml
exclude:
  files:
    - ".*_generated\\.go$"
  types:
    - "example\\.com/project/provider\\.Service$"
```

- Expected: no diagnostics from `provider/generated.go` (file is excluded).
- Expected: `provider.Service` is ignored by both rules due to `exclude.types`.

`exclude.files` skips entire files, and `exclude.types` skips matching types. Both rules ignore anything matched by these exclusions. This is useful for generated code, mocks, or legacy areas you intentionally exclude.

## Requirements

- Go 1.22+
- https://taskfile.dev/
- https://golangci-lint.run/

## Development Commands

- `task lint` - Run linter
- `task test` - Run all tests
- `task build` - Build custom golangci-lint with ifaceguard linter and ifaceguard binary
- `task fmt` - Format Go code
- `task check` - Run lint and test (full validation)
- `task test:custom` - Test custom `bin/custom-gcl` linter on this codebase. This is a "negative" test that should return errors, as the test data intentionally violates the rules.
