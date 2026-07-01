# Contributing to rubix

## Requirements

* [Go](https://go.dev) (stable — version from `go.mod`)
* [just](https://github.com/casey/just)
* [golangci-lint](https://golangci-lint.run/welcome/install/) (for `just lint`)
* [gremlins](https://github.com/go-gremlins/gremlins) (for mutation testing)

A C toolchain is only needed for the optional Ebiten visualizer (`just build-gui`).

## Development workflow

```
just build      # build the headless CLI
just build-gui  # build with the Ebiten visualizer (-tags ebiten, needs cgo)
just test       # run unit tests
just test-gui   # run tests for the GUI build
just lint       # golangci-lint
just fmt        # golangci-lint fmt (gofumpt + goimports)
just ci         # lint + test + build
just tidy       # go mod tidy
just view       # run the self-driving visualizer
just demos      # regenerate demo assets
```

Run `just --list` to see every recipe. Run `just ci` (lint + test + build) before each commit. CI runs the same gate on every push to `trunk` and every pull request targeting `trunk`.

## Conventions

The CLI entrypoint and Ebiten GUI structure is shared across the tool family
(unum is the CLI reference; rubix/vivarium the GUI references). See
[CONVENTIONS.md](CONVENTIONS.md).

## Project layout

rubix is both a CLI and a reusable cube library.

```
pkg/cube/        public cube library
cmd/rubix/       CLI entry point (solve, replica, scramble, view, …)
internal/        implementation packages (solver, robot, gui, cli)
docs/            documentation
```

## Commit style

```
type(scope): short imperative description
```

Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`. No period at the end of the subject line; keep it under 72 characters.

## Releases

Releases are triggered by pushing a semver tag — maintainers only. A GitHub Actions workflow runs GoReleaser to build the binaries (and the macOS GUI cask) and update the Homebrew tap; it requires the tap app credentials configured as repository secrets.
