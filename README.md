# rubix

> *rubix* — the cube as data, solved in code.

[![CI](https://github.com/danielriddell21/rubix/actions/workflows/ci.yaml/badge.svg)](https://github.com/danielriddell21/rubix/actions/workflows/ci.yaml)
[![codecov](https://codecov.io/gh/danielriddell21/rubix/graph/badge.svg)](https://codecov.io/gh/danielriddell21/rubix)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=danielriddell21_rubix2&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=danielriddell21_rubix2)
[![Go Reference](https://pkg.go.dev/badge/github.com/danielriddell21/rubix/pkg/cube.svg)](https://pkg.go.dev/github.com/danielriddell21/rubix/pkg/cube)
[![Go 1.26](https://img.shields.io/badge/go-1.26-blue)](https://go.dev)
[![MIT License](https://img.shields.io/badge/licence-MIT-green)](LICENSE)

A Rubik's cube solver in Go. The cube lives as plain data in a headless core, solved nine different ways, with an optional 3D visualizer and a LEGO Mindstorms EV3 driver.

A port and extension of two "Coding Adventure" videos by Sebastian Lague: [Solving the Rubik's Cube](https://www.youtube.com/watch?v=fy0HRViXZnE) and [I Tried Optimizing my Rubik's Cube Solver](https://www.youtube.com/watch?v=Eysf6-E3ino).

## Commands

| Command | Description |
|---|---|
| `rubix solve` | Solve a cube, from a facelet string or the robot |
| `rubix solvers` | List the available solvers in video order |
| `rubix scramble` | Generate a scramble and its facelet string |
| `rubix verify` | Validate a facelet string |
| `rubix view` | Open the self-driving 3D visualizer |
| `rubix scan` | Scan a real cube with the EV3 robot |
| `rubix gen-tables` | Precompute and cache the prune tables |

## Install

### Homebrew
```sh
brew install danielriddell21/tap/rubix
brew install --cask danielriddell21/tap/rubix
```

### Go install
```sh
go install github.com/danielriddell21/rubix/cmd/rubix@latest
```

### From source
```sh
git clone https://github.com/danielriddell21/rubix
cd rubix
go run ./cmd/rubix solvers
```

Requires Go 1.26+. The 3D visualizer is behind the `ebiten` build tag:

```sh
go run -tags ebiten ./cmd/rubix view
```

## Library

`pkg/cube` is the public package — the cube model, moves and facelet handling — and is documented on [pkg.go.dev](https://pkg.go.dev/github.com/danielriddell21/rubix/pkg/cube).

## Documentation

Full documentation lives in the [rubix wiki](https://github.com/danielriddell21/rubix/wiki) — every command and flag, the nine solvers and their solve rates, the architecture, and where this deviates from the videos.
