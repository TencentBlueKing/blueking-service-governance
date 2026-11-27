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
* When a unit test needs a store, use `testutil.NewStore(GinkgoT(), XxxMeta)`; when it needs the database handle, use `testutil.NewDB(GinkgoT())`. See `pkg/extension/bscpcfg/model/store_test.go` for the pattern.
* When a unit test needs a service or other FX-provided dependency, construct and inject it via the fx framework: use the package's `FxModule` with `fxtest.New` + `fx.Populate`. Do **not** call constructors like `NewService(...)` directly in tests.
* Do **not** mock stores (no stub/fake store, no `mockey.Mock` on `*store.GenericStore` methods). Operate against the test database and assert persisted data. Only mock store methods after confirming with the user, and only when hitting the real DB is prohibitively expensive.
* Prefer using `dbfactory` helpers (e.g., `dbfactory.TrpcApplication`, `dbfactory.Env`) to create test resources, rather than manually inserting records into the database.

## Coding style

* For Go, follow the official Go style guide.
* Prefer using [`samber/lo`](https://github.com/samber/lo) where it simplifies code without harming readability. Common helpers already used in this codebase include:
  - Pointer helpers: `lo.ToPtr`, `lo.FromPtr`, `lo.FromPtrOr`, `lo.EmptyableToPtr`
  - Collection helpers: `lo.Map`, `lo.Filter`, `lo.FilterMap`, `lo.Find`, `lo.MapEntries`, `lo.Keys`, `lo.Values`, `lo.Uniq`, `lo.Contains`
  - Other useful helpers: `lo.Coalesce`, `lo.Ternary`, `lo.Must`, `lo.If` / `lo.Else`
* For logging conventions, refer to [`README.md#日志使用`](README.md#日志使用): use `pkg/common/logging`, prefer passing a real `context.Context`, use `NoContext` APIs only when no real context is available, and use `*Attrs` APIs for `slog.Attr` structured fields.
* Wrap errors with [`github.com/pkg/errors`](https://github.com/pkg/errors) (`errors.Wrap` / `errors.Wrapf`). Do not use `fmt.Errorf` to wrap errors; `fmt.Errorf("%w: %w", err, sentinel)` is also disallowed. For taskq unrecoverable failures, wrap the sentinel: `errors.Wrap(taskq.ErrStopRetry, "reason")` or `errors.Wrapf(taskq.ErrStopRetry, "reason: %v", err)`.
* **NEVER `return err` bare when propagating an error upwards** — every layer adds what it was doing, with `Wrapf` folding in identifiers, so the message chain alone pinpoints the failure. Two exceptions: sentinels matched upstream by `errors.Is` still get wrapped (wrapping preserves `errors.Is`), and skip the wrap when the layer directly below already located the failure.
* Never log an error and also return it, or it shows up twice — log only where the error is genuinely swallowed, otherwise let the final handler log it once.
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
* return errors as described in "Returning errors from handlers" below
* add Swagger annotations and run `make apidocs`
* In Swagger annotations, only declare `@Failure 400`; do NOT add other error codes such as 401, 403, 404, or 5xx (e.g. a handler should have just `@Success 200` and `@Failure 400 {object} bkerrs.GinErrorOutput`)
* `make build` to build the binary

#### Returning errors from handlers

Applies whenever you write or modify a handler. Full reference: [`pkg/common/bkerrs/README.md`](pkg/common/bkerrs/README.md).

**Do NOT copy `bkerrs.Wrap(err, bkerrs.ErrCodeInternalServerError, "...")` from existing handlers.** It is a legacy pattern that appears hundreds of times; this rule overrides "be consistent with nearby code". Wrapping as 500 overrides the error exit's classification: a store "not found" caused by a user-supplied ID becomes a 500 instead of a 404, and the user only sees a generic "internal server error".

Pick the first case that applies:

1. `err` already carries an API error (from `ginutils.BindURI` / `BindJSON`, `perm.Validate*`, or any function returning `*bkerrs.Error`): `bkerrs.AbortWithErr(c, err)`. Do not wrap it again.
2. **Default** for errors returned by a service or store call: `bkerrs.AbortWithErr(c, errors.Wrapf(err, "get env %s", envID))`. You do not choose a code: the error exit returns 404 / 409 for store "not found" / "already exists" and 500 for everything else.
3. Some domain errors should become a specific 4xx: same as case 2, plus rules, e.g. `bkerrs.WhenIs(model.ErrUnknownSwitch, bkerrs.ErrCodeInvalidArgument, i18n.Msg("unknown cfg switch key"))`. Check the domain's `apierrs` package (e.g. `pkg/core/env/apierrs`) for existing rules first. Only map errors caused by user input; never map downstream failures to 4xx.
4. The whole error means the user's input is invalid or the resource they asked for does not exist: `bkerrs.AbortInvalidArg(c, err, i18n.Msg("invalid version ID"))` / `bkerrs.AbortNotFound(c, err, i18n.Msg("..."))`.
5. `bkerrs.AbortInternal(c, errors.Wrap(err, "list bscp user bizs"))` only when the failure can never be the user's fault, e.g. a downstream API call or a lookup by a system-generated ID. Ask: "could this fail because the user passed a bad ID?" If yes, use case 2.

Global rules (`ginutils.GlobalErrorRules`, e.g. BSCP disabled) apply automatically; do not map them in handlers. Model, service and store code returns domain errors (sentinels or typed errors) wrapped with `errors.Wrap`; the handler decides the code. Do not add `bkerrs` to new non-handler code, even though a few legacy packages still use it.

Messages come in two kinds. A shortcut's third argument is always user-facing; `AbortInternal` has none because internal errors always show a generic message, so attach developer context with `errors.Wrap` as in case 2.

* **User-facing** (`AbortInvalidArg` / `AbortNotFound` msg, `WhenIs` / `Errorf` / `Wrapf` format): a complete sentence describing the problem to the user, e.g. `i18n.Msg("invalid version ID")`, never a developer phrase like `"parse version ID"`. Always mark it with `i18n.Msg`, then run `make i18n` and fill in `pkg/common/i18n/locales/zh-cn.po`.
* **Developer context** (`errors.Wrap` / `Wrapf`, `NewDetail` message): a short "what was being done" phrase with identifiers via `Wrapf`. Never mark it with `i18n.Msg`; users never see it.

Never write `bkerrs.New(code, err.Error())`: it drops the error chain and cannot be translated. Pass dynamic values as format args, never pre-format a message with `fmt.Sprintf`.

#### Database model & migrations

* Models are persisted with `GenericStore` (`pkg/store`): define a `store.Meta` and named `Opt`s in the model package, and get the store with `store.Of(db, XxxMeta)`; do not define a per-model Store interface. Read `design_notes/generic_store.md` first, and use `pkg/core/app/store.go` as a full example (hooks, unique keys, aggregation).
* Domain code receives `*mongo.Database` explicitly instead of accessing `storereg.G()`. Functions that read or write more than one model must list those models in their doc comment, since the signature no longer shows them.
* When adding new migrations, read the guide in `README.md`, specifically the "数据库迁移" section
* ALWAYS add a comment for extra indexes in the store source file
* Secret fields (passwords, tokens, sensitive env var values) on models that may be written to operation records MUST carry an `audit` struct tag (`audit:"mask"` or `audit:"mask,if=IsSensitive"`), see `pkg/misc/audit/sanitize.go`.
* Before writing code that touches multi-tenancy (`Meta.TenantScoped`, `store.TenantAll` / `store.TenantOf`, tenant in ctx, taskq envelope, tests of tenant scoped collections), read `design_notes/tenant_isolation.md`

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
