# Project Guidelines

NO BACKWARDS COMPATIBILITY, NO DEPRECATION! THIS PROJECT IS IN ACTIVE DEVELOPMENT!

## Overview

`ifaceguard` is a linter that checks architectural properties of interface usage in Go code:

1. **DIP (Dependency Inversion Principle):** the interface as a contract should "belong" to the consumer side (or a dedicated contract package), not the implementation side.
2. **Explicit compile-time assertion of contract implementation:** the package that declares the implementation type places a compile-time assertion of interface implementation.

## Behavior

1. Standalone executable
2. golangci-lint (module plugin)

More details in the `README.md`

## Available Commands

- `task lint` - Run linter
- `task test` - Run all tests
- `task build` - Build custom golangci-lint with ifaceguard linter and ifaceguard binary
- `task fmt` - Format Go code
- `task check` - Run lint and test (full validation)
- `task test:custom` - Test custom `bin/custom-gcl` linter on this codebase. This is a "negative" test that should return errors, as the test data intentionally violates the rules.

## Rules

1. MUST use English for all code and comments.
2. MUST execute `task lint` before finalizing.
3. Go 1.24+