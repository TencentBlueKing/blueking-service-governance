## Purpose

* These brief instructions help LLM agents navigate working in the bkms-cli repo.
* **Humans** are always responsible for changes being proposed and must pre-review all agentic work before turning it into a PR.

## Context

You are in the bkms-cli repo, helping implement features, fix bugs, and refactor existing code.

Source files in this repo can be very long. Check their size to consider if
you really need to read the entire thing.

If tools such as `rg` (ripgrep), `gh`, `jq`, or `prek` are not found, ask
the user to install them. ALWAYS prefer using `rg` rather than `find` or `grep`.

## Source code

* bkms-cli is a Cobra-based CLI tool implemented in Go.
* `main.go` is the entry point; it creates a cancellable context with signal handling (Ctrl+C).
* Subcommands are defined under `cmd/`, each subcommand in its own directory (e.g. `cmd/app/`, `cmd/env/`).
* The root command is in `cmd/root/root.go`, where all subcommands are registered and auth/init logic lives.
* Business logic and shared packages live under `pkg/`.
* Unit tests are placed in each package's directory using Ginkgo + Gomega, following Go conventions.
* E2E tests live in `test/e2e/` and exercise the built CLI binary directly.
* When writing unit tests, refer to `pkg/config/config_test.go` for guidance on test structure, temporary config setup, and usage of Ginkgo/Gomega.

### Adding a new subcommand

1. Create a new directory under `cmd/` (e.g. `cmd/myfeature/`).
2. Define the command entry file following the pattern in `cmd/env/env.go` (a `NewCmd()` function returning `*cobra.Command`).
3. Register the new command in `cmd/root/root.go` via `rootCmd.AddCommand(...)`.
4. Place API call logic in `pkg/client/` and business logic in `pkg/handler/`.
5. If the command should work without authentication, set `Annotations: map[string]string{cmdutil.SkipAuthAnnotationKey: "true"}`.

### app create subcommand design

* Users provide a YAML spec file via `bkms-cli app create -f app.yaml`.
* The YAML is parsed into `AppCreateSpec` (defined in `pkg/handler/app/types.go`).
* Validation uses `go-playground/validator` with custom tags (`app_name`, `envvar_key`) and struct-level validators.
* CLI-side validation is intentionally lightweight (required fields + format); full business rules are enforced by the backend.
* App ID can be user-specified (`id` field in YAML) or auto-generated (name + random suffix from backend API).
* Request body conversion uses json-tagged intermediate structs (`pkg/handler/app/request.go`) serialized via `encoding/json`.

## Coding style

* For Go, follow the official Go style guide.
* We use golangci-lint (v2, config in `.golangci.yaml`) to lint and format files.
* Be consistent with existing nearby code style unless asked to do otherwise.
* NEVER leave trailing whitespace on any line.
* ALWAYS preserve the newline at the end of files.
* Every new source file starts with the MIT license header. Copy it verbatim from an existing file of the same type, for example `main.go`. See "License headers" in the repository root [`AGENTS.md`](../AGENTS.md) for placement rules and exceptions.
* Use `console.Info/Tips/Error/Debug` for CLI output rather than bare `fmt.Println`.
* For client interface tests, prefer generated mockery mocks over handwritten fake clients.

### sub-command design guidelines

* For agent-facing command references, add concise Markdown files under `.agents/prompts/`.
    - Start with a short feature summary, then common workflows, then per-subcommand examples. Prefer "short explanation paragraph + command block" for workflow sections, and avoid exhaustive flag tables unless explicitly requested.
- For edit-like commands, support both file input and literal content input when practical. Treat `--file` and `--file-content` as mutually exclusive.

## Testing

### Unit tests

* Unit tests use **Ginkgo v2 + Gomega**.
* Run all unit tests: `make test`
* Run a specific package: `./bin/ginkgo run ./pkg/config/`
* Test files follow the pattern `*_test.go` and `*_suite_test.go` alongside source files.

### E2E tests

* E2E tests live in `test/e2e/` and use Ginkgo v2 + Gomega.
* They exercise the compiled CLI binary directly via `framework.CLI`.
* Before running, build the E2E binary and set required env vars: `make e2e-go-test`
* The E2E framework auto-loads `test/e2e/.env` for environment configuration.

## Common workflows

* After editing Go code, run: `make lint`
* After editing Go code, run unit tests: `make test`