# Cove

Cove is a Wails v3 desktop application using Go, React, TypeScript, and pnpm.

## Getting Started

Install dependencies:

```sh
pnpm --dir frontend install
go mod tidy
```

Run in development mode:

```sh
wails3 dev -config ./build/config.yml
```

Build for production:

```sh
wails3 build
```

## Project Structure

- `main.go` embeds frontend assets and starts the assembled app.
- `internal/app` wires Wails options, services, and the main window.
- `internal/services` contains Wails-bound backend services.
- `internal/domain` contains Wails-independent business types.
- `internal/platform` wraps system-specific helpers.
- `frontend/src/app` contains the React application shell.
- `frontend/src/shared` contains reusable frontend utilities and API wrappers.
