## Purpose

* These brief instructions help LLM agents navigate working in the bkms-server repo.
* **Humans** are always responsible for changes being proposed and must pre-review all agentic work before turning it into a PR.

## Context

You are in the bkms-server repo, helping implement features, fix bugs, and refactor existing code.

Source files in this repo can be very long.  Check their size to consider if
you really need to read the entire thing.

If tools such as `rg` (ripgrep), `gh`, `jq`, or `prek` are not found, ask
the user to install them. ALWAYS prefer using `rg` rather than `find` or `grep`.

## Source code

* bkms-server is a Gin REST API service implemented in Go.
* API definitions and request/response types are located in each feature package's router, handler, and serializer packages.
* Unit tests are placed in each package's directory, following Go conventions.
* Some design notes can be found in `design_notes/`.
* When writing tests, always refer to `pkg/extension/component/evaluate_test.go` for guidance on code structure and the usage of common utils and db factories.
* Test context descriptions (`Describe`, `Context`, `It`, etc.) MUST be written in English. Comments inside test bodies may use Chinese.
* Prefer using `FxModule` (e.g., `workload.FxModule`, `polaris.FxModule`) with `fx.Populate` to inject dependencies in tests, rather than manually constructing stores or services.
* Prefer using `dbfactory` helpers (e.g., `dbfactory.TrpcApplication`, `dbfactory.Env`) to create test resources, rather than manually inserting records into the database.

## Coding style

* For Go, follow the official Go style guide.
* For logging conventions, refer to [`README.md#日志使用`](README.md#日志使用): use `pkg/common/logging`, prefer passing a real `context.Context`, use `NoContext` APIs only when no real context is available, and use `*Attrs` APIs for `slog.Attr` structured fields.
* We use golangci-lint to lint and format our files.
* Be consistent with existing nearby code style unless asked to do otherwise.
* NEVER leave trailing whitespace on any line.
* ALWAYS preserve the newline at the end of files.
* Every new source file starts with the MIT license header. Copy it verbatim from an existing file of the same type, for example `main.go`. See "License headers" in the repository root [`AGENTS.md`](../AGENTS.md) for placement rules and exceptions.

### Running our tests

* ALWAYS use `ginkgo`, the tests are `ginkgo` based.
* ALWAYS use `-gcflags="all=-l -N" --cover --coverprofile cover.out` flags.
* When running tests, prefer specify the module path like `./bin/ginkgo run ./pkg/arrangement/values/` is possible.
* run `make test` to run all tests.

### Common workflows

* After editing Go code, run: `make lint`
* After editing Go code, run tests: `./bin/ginkgo {TEST_FILE}`
* After editing scripts under `pkg/extension/component/devmode/assets`, update the corresponding tests under `pkg/extension/component/devmode/assets/tests`, run `just lint && just test` to verify

#### Add new REST APIs

* make changes to the router, handler, and serializer files in each feature package
* keep handler code simple, place complex logic in other modules
* add Swagger annotations and run `make apidocs`
* `make build` to build the binary

#### Database model & migrations

* When adding new store for new models, use `pkg/core/app/store.go` as reference
* When adding new migrations, read the guide in `README.md`, specifically the "数据库迁移" section
* ALWAYS add a comment for extra indexes in the store source file

### API Tests

#### Pre-requirement: Start local bkms-server service

- compile the latest binary: `make build`, IMPORTANT: only run if `.go` files updated
- restart the server: cd `tests/`, `just down && just up-locbin`

### Run bruno tests

- IMPORTANT: before running tests, re-compile and restart the service if necessary
- In `tests/apis`
- Test directory recursively: `bru run --env-file ./environments/local.bru -r {DIR}`
- Test single file: `bru run --env-file ./environments/local.bru -r {FILE}`

### New bruno test cases guideline

- Use `tests/apis/app_config_files/set_content_of_normal_should_succ` as a reference
- Test cases within the same sub-folder can have sequential dependencies
- Tests in different directories should remain isolated
