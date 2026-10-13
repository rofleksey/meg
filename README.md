# meg

Reusable Go infrastructure for services and command-line programs. Packages have
separate imports so value helpers do not pull the telemetry stack into a build.

| Package | Responsibility |
| --- | --- |
| [ptr](ptr) | Pointers and nil-to-zero values |
| [logging](logging) | JSON/text `slog` with credential-key redaction |
| [operation](operation) | `unolog` completion events through `slog` |
| [otelmetrics](otelmetrics) | Optional OpenTelemetry metrics over OTLP/HTTP |
| [httpx](httpx) | Validated, consistent HTTP request IDs |
| [ratelimit](ratelimit) | Bounded per-key token-bucket budgets |
| [process](process) | Bounded execution of trusted commands without a shell |

Applications own routes, error envelopes, metric names, authorization, command
allowlists, and domain policy. Supported integration targets are Akio Main
and Akio Agent.

## Use

```sh
go get github.com/rofleksey/meg@v0.1.2
```

Requires Go **1.26.6** or newer. CI tests Go 1.26.6 and 1.27.1 on Linux and
Go 1.27.1 on Windows. Keep the minimum Go version separate from the toolchain
used to develop a consuming application.

Use the packages directly or through an application-owned adapter:

```go
logger := logging.New("json", os.Stderr)
runtime := operation.NewRuntime(logger)
handler := httpx.RequestID(applicationHandler)
```

`httpx.RequestID` validates an inbound `X-Request-ID` or creates one. Downstream
handlers receive the same validated value in the request header, context
(`httpx.RequestIDFrom`), and response header. The original request headers are
not mutated. Apply it before validation and request logging.

`logging` redacts credential-like keys, including nested groups and lazy
`slog.LogValuer` results. This is defense in depth: messages, arbitrary structs,
and values under harmless keys are not a general-purpose secret scanner.
Callers must omit secrets and unrestricted data. `operation.RecordError` records
a safe error kind instead of a dependency's raw error message. One owner starts,
enriches, and completes each operation.

`ratelimit.NewLimiter` creates an independent budget per key with bounded key
count. Budgets are local to one process; global limits across replicas need an
application-owned shared or upstream limiter. Rejections return a duration;
HTTP callers must round it **up** to whole seconds for `Retry-After`.

`otelmetrics.Open` reads standard `OTEL_*` metrics settings. Export is disabled
unless an endpoint is configured or `OTEL_METRICS_EXPORTER=otlp` is explicit;
`none` disables it. The supplied service name identifies the application.
Callers decide how to handle initialization errors and must give `Shutdown` a
bounded context. Applications own instruments and bounded metric attributes.

`process.Request` requires a trusted executable, argument list, working
directory, positive timeout, and per-stream output limit. The runner drains
stdout/stderr while retaining at most that limit from each stream. An exceeded
limit returns `ErrOutputLimit`; when execution also fails, `errors.Is/As` still
find the execution/cancellation cause. Treat captured output as untrusted data.
On Linux, cancellation signals the process group, with a bounded grace period
before forceful cleanup. Remaining group members are also killed after normal
child exit, before the leader is reaped, so cleanup cannot target a reused group
ID. Descendants that start a new session escape that group; inherited output
handles still have a bounded drain. Other platforms terminate only the direct
child and require application-owned infrastructure for process-tree isolation.

## Develop and verify

Install **golangci-lint v2.14.0**. Standalone staticcheck is pinned by the Go tool
directive. The linter configuration lists every enabled rule explicitly.

```sh
make format                    # gofmt and goimports
make check                     # lint, formatting, vet, staticcheck
make test race build           # behavior, races, compilation
go mod tidy                    # review any module/checksum changes
```

[AGENTS.md](AGENTS.md) contains contributor rules and ownership boundaries. Tests
use owned subprocesses, local test servers, and in-memory telemetry; they do not
need application deployments or production services. CI also checks that module
files remain tidy.

## Release and consumer updates

1. Run the complete verification gate and review the API/behavior diff. Test the
   candidate in supported consumers with a temporary Go workspace or local
   replacement; keep these local overrides out of their commits.
2. Commit and push the library, wait for CI, then publish a new immutable semantic
   version tag. Never move an existing release tag.
3. Update supported consumers to that tag, tidy their module files, and run their
   own generation, lint, test, integration, and build checks without local overrides.
4. Commit/push consumers and verify CI. Artifact publication and deployment are
   separate operations; deploying requires its own authorization.

Preserve public contracts within compatible releases and record material behavior
changes here. Version 0.1.2 fixes request-ID propagation, lazy-value redaction,
subprocess output/cancellation handling, and telemetry initialization bounds;
it also adds the contributor guide and reproducible lint/CI gates.
