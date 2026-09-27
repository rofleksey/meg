# meg

Reusable Go building blocks for services and command-line programs. Packages
have separate imports so small applications do not need the telemetry stack.

| Package | Purpose |
| --- | --- |
| [ptr](ptr) | Pointers and nil-to-zero values |
| [logging](logging) | JSON/text `slog` with key-based credential redaction |
| [operation](operation) | `unolog` operation lifecycle through `slog` |
| [otelmetrics](otelmetrics) | Optional OpenTelemetry metrics export over OTLP/HTTP |
| [httpx](httpx) | Validated HTTP request IDs |
| [ratelimit](ratelimit) | Bounded per-key token-bucket budgets |
| [process](process) | Bounded subprocess execution without a shell |

The library does not define application metric names, routes, error envelopes,
commands, or policy. Applications construct those at their own boundaries.
`process.Request` must be built from trusted, locally allowlisted inputs.
`logging` redaction is defense in depth; callers should omit secret values
from telemetry entirely.

Install a tagged version with `go get github.com/rofleksey/meg@v0.1.1`.
Run `go test ./...` and `go vet ./...` before releasing a change.
