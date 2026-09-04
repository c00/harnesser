# Agent Instructions

This is the canonical entry point for coding agents working in this repository. Use the task-specific guides in `.agents/` before implementing related changes.

## Testing

- Always use the `github.com/stretchr/testify` library for assertions and requirements.
- Use `assert` for non-fatal failures and `require` for failures that should stop test execution immediately.
- When using contexts in tests, use `t.Context()` as the root.
- Table tests are preferred.
- Test functions as purely as possible.

## Logging

- Use the `log/slog` package for logging.
- Initialize loggers with a `package` attribute for context.
- Do not use a global package logger that bypasses the custom logger.
- Add the logger to the struct if applicable, for example: `SomeStruct{logger: slog.Default().With("package", "somestruct")}`.
- If no struct is available, use a helper function that returns the logger.
- Use appropriate log levels: `Debug`, `Info`, `Warn`, and `Error`.
- Long-running functions such as `Run(ctx)` should log an info message at start and finish.
- Do not log errors that are also returned.
- When a context is available, use context-aware methods such as `WarnContext(ctx, ...)`.

## Git diff and gofmt

Do not do git actions unless specifically asked to. Do not do gofmt actions unless specifically asked to.
