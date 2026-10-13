# meg contributor guide

meg provides reusable Go infrastructure for services and command-line programs.
Keep packages independently importable and focused on their public contracts.
Current integration targets are Akio Main and Akio Agent.

## Ownership and API design

- `logging` owns slog construction and credential-key redaction; `operation`
  bridges unolog completion events; `otelmetrics` configures optional metrics
  export. Applications own event/metric names, safe fields, and fail-open policy.
- `httpx` owns protocol mechanics, `ratelimit` owns bounded token-bucket state,
  `process` owns trusted executable lifecycle/output, and `ptr` owns optional
  value helpers. Application routes, error envelopes, authorization, command
  allowlists, and domain policy belong in consumers.
- Add shared behavior when a supported consumer demonstrates a need. Preserve
  current exported APIs where practical; document intentional compatibility
  changes and update the supported consumers together.
- Prefer the standard library and maintained primitives to custom frameworks.
  Keep simple packages free of telemetry dependencies.
- Put shared exported contracts in `types.go` when it improves navigation;
  keep private helpers beside their behavior. Avoid redundant subtree guides.
- Use contexts for cancellation, wrap dependency errors with useful context,
  and preserve `errors.Is/As`. Document result/error precedence and platform
  limits when callers must make decisions from them.

## Resource and telemetry contracts

- Bound retained state, captured output, startup/cleanup waits, and export work.
  Own and release partially initialized resources on every error path.
- Process requests use trusted executable/argument inputs without an implicit
  shell. Preserve cancellation and platform-specific descendant cleanup rules.
- Redaction is defense in depth, not permission to log arbitrary data. Keep
  credentials and unrestricted payloads out of test artifacts and telemetry.
- Resolve lazy slog values before examining nested groups. Preserve the slog
  handler contract, including derived handlers and concurrent use.
- Operation completion has one owner. Use bounded metric dimensions and let
  application boundaries decide how to report telemetry setup/export failures.

## Verification

- Use the Makefile targets and pinned tools. golangci-lint v2 uses an explicit
  linter list, gofmt/goimports, and narrowly explained exceptions.
- Run `make check test race build` before release. Tests should prove observable
  behavior, failure handling, and concurrency contracts rather than mirror code.
- Keep process tests self-contained and deterministic, with deadlines and cleanup
  for owned children. Never invoke production services or arbitrary host commands.
- CI verifies the minimum and current supported Go versions, plus Windows tests
  for portable packages and the documented Windows process behavior.
- Validate consumer changes against the candidate library before publishing.
  Temporary workspace/replace directives and generated artifacts stay uncommitted.

## Releases

Update README usage and compatibility notes when behavior changes. Keep minimum
Go support aligned with tested consumers and CI; upgrade dependencies for an
identified reason, not as an unrelated sweep. Publish an immutable version tag
only after verification, then update supported consumers to that tag and verify
their own checks. Pushing releases is not deployment authorization.
