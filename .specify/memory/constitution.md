<!--
Sync Impact Report
==================
Version: 0.0.0 → 1.0.0
Change Type: Initial constitution creation
Modified Principles: N/A (all new)
Added Sections:
  - Core Principles (5 principles)
  - Go Language Standards
  - Development Workflow
  - Governance
Templates Status:
  ⚠ .specify/templates/plan-template.md - pending review
  ⚠ .specify/templates/spec-template.md - pending review
  ⚠ .specify/templates/tasks-template.md - pending review
  ⚠ .specify/templates/commands/*.md - pending review
Follow-up TODOs:
  - Review template files for alignment with new principles
  - Update agent guidance files if needed
-->

# Worktree Manager (wtm) Constitution

## Core Principles

### I. Go Idiomatic Code (NON-NEGOTIABLE)

All code MUST follow official Go conventions and best practices:

- Use `gofmt` for formatting - no exceptions
- Follow [Effective Go](https://go.dev/doc/effective_go) patterns
- Adhere to [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use standard library first; external dependencies require justification
- Error handling is mandatory - every error must be explicitly checked and handled

**Rationale**: Go's strength lies in its simplicity and consistency. Idiomatic code ensures maintainability, readability, and leverages the ecosystem effectively.

### II. Comprehensive Testing (NON-NEGOTIABLE)

Testing is non-negotiable and MUST cover all code paths:

- Minimum 80% code coverage required for all packages
- Tests written before or alongside implementation (Test-Driven Development encouraged)
- Every function MUST have tests for: happy path, error conditions, edge cases
- Integration tests MUST verify git operations with real repositories
- Table-driven tests preferred for multiple scenarios
- Test isolation enforced: use `t.TempDir()`, reset global state, no test interdependencies

**Rationale**: Git operations are complex and error-prone. Comprehensive testing prevents regressions and ensures reliability across diverse repository states.

### III. User-Centric Error Messages

Error messages MUST be actionable and guide users toward resolution:

- Clearly state what went wrong
- Suggest specific next steps or commands
- Include relevant context (file paths, current state)
- Example: `"file not found in shared storage: %s\nUse 'wtm persist list' to see available files"`
- Never expose internal implementation details or stack traces to users

**Rationale**: Users should never feel stuck. Clear errors reduce frustration and support burden.

### IV. Command Consistency (Cobra Patterns)

All commands MUST follow consistent patterns for predictable UX:

- Use Cobra framework for all CLI commands
- Subcommands group related operations (e.g., `persist add/list/remove`)
- Flags use consistent naming: `--force` for overwrites, `--link` for symlinks, `--all` for bulk operations
- Command structure: `wtm <command> [subcommand] <args> [flags]`
- Help text MUST include usage examples for complex commands
- Return appropriate exit codes: 0 for success, 1 for user errors, 2+ for system errors

**Rationale**: Consistency reduces cognitive load and makes the tool intuitive for users familiar with modern CLI conventions.

### V. Git Safety & Worktree Integrity

All git operations MUST preserve repository integrity and prevent data loss:

- Validate repository state before mutations
- Use `GIT_DIR` environment variable for worktree operations
- Never destructively modify the `.git` directory
- Preserve file permissions when copying or persisting
- Require explicit `--force` flag for any destructive operations
- Verify symlink targets are accessible before creating links
- Fall back gracefully when git operations fail

**Rationale**: Git repositories contain valuable work. Safety mechanisms prevent accidental data loss and corruption.

## Go Language Standards

### Code Organization

- One package per directory; avoid package name stuttering
- Keep `main` package minimal (entry point only)
- Group related functionality in `cmd` package
- Use descriptive package names (avoid generic names like `util`, `common`, `helper`)

### Error Handling

- Wrap errors with `fmt.Errorf("context: %w", err)` to maintain error chain
- Return errors as last return value
- Fail fast on invalid input
- Provide context in wrapped errors

### Function Design

- Single Responsibility Principle: one function does one thing
- Limit functions to ~50 lines when possible
- Use structs for functions with >3 parameters
- Document all exported functions with godoc comments

### Naming Conventions

- `camelCase` for unexported names
- `PascalCase` for exported names  
- `snake_case` only for test file names
- Use intention-revealing names

## Development Workflow

### Pre-Commit Requirements

Before committing, developers MUST:

1. Run `go fmt ./...` - code must be formatted
2. Run `go vet ./...` - no vet warnings allowed
3. Run `go test ./...` - all tests must pass
4. Ensure new code has tests written
5. Update documentation if behavior changes

### Code Review Standards

All pull requests MUST:

- Include tests for new functionality
- Maintain or increase code coverage
- Update relevant documentation
- Pass all CI checks
- Have clear, descriptive commit messages
- Follow conventional commit format: `type(scope): description`
  - Types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`

### Testing Requirements

- Unit tests for all public functions
- Integration tests for git operations
- Test isolation via `t.TempDir()` and state resets
- Descriptive test names: `TestFunctionName/scenario_description`
- AAA pattern: Arrange, Act, Assert

## Governance

This constitution supersedes all other development practices and guidelines. All contributors MUST:

- Follow these principles in all code contributions
- Verify PR compliance before review
- Justify any complexity or deviation from simplicity
- Update constitution when fundamental practices change

### Amendment Process

Constitution changes require:

1. Proposal with clear rationale
2. Discussion and consensus among maintainers
3. Version bump following semantic versioning:
   - MAJOR: Breaking changes, principle removals/redefinitions
   - MINOR: New principles added, expanded guidance
   - PATCH: Clarifications, typos, non-semantic refinements
4. Update date stamp
5. Sync check of all dependent templates and documentation

### Compliance Review

- Constitution compliance checked during all code reviews
- Automated checks via CI where possible (formatting, tests, coverage)
- Regular audits to ensure ongoing adherence

**Version**: 1.0.0 | **Ratified**: 2025-11-24 | **Last Amended**: 2025-11-24
