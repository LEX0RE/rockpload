# Contributing

Thanks for your interest in contributing to rockpload.

## Contribution Policy

All contributions are welcome, including bug fixes, refactors, and new features.

## Prerequisites

- Go 1.26.1
- CGO toolchain
- Linux GCC toolchain for Linux builds
- MinGW toolchain for Windows cross-compilation when building from Linux:
  - x86_64-w64-mingw32-gcc
  - x86_64-w64-mingw32-g++

## Local Setup

1. Clone the repository.
2. Create your local environment file:

```bash
cp .env.example .env
```

3. If you use custom Epic OAuth, set values in `.env`:

```bash
EPIC_AUTH_MODE=custom
EPIC_CLIENT_ID=
EPIC_CLIENT_SECRET=
```

By default, `EPIC_AUTH_MODE=launcher` is recommended.

## Build

```bash
scripts/build.sh
```

Current support target:

- Linux amd64
- Windows amd64
- macOS is not supported

## Code Quality

Before opening a PR, run:

```bash
go fmt ./...
go vet ./...
go build ./...
```

## Local State

rockpload stores runtime state in the user cache directory:

- `~/.cache/rockpload/.rltoken`
- `~/.cache/rockpload/.uploaded`
- `~/.cache/rockpload/rockpload.lock`

On Windows, this maps to `%LOCALAPPDATA%\\rockpload\\...`.

## Pull Requests

1. Fork the repository
2. Create a feature branch from `master`
3. Commit focused changes with clear messages
4. Open a pull request using the PR template

No test suite exists yet. Adding tests is a high-impact contribution.
