# rubix

[![CI](https://github.com/danielriddell21/rubix/actions/workflows/ci.yaml/badge.svg)](https://github.com/danielriddell21/rubix/actions/workflows/ci.yaml)
[![codecov](https://codecov.io/gh/danielriddell21/rubix/branch/trunk/graph/badge.svg)](https://codecov.io/gh/danielriddell21/rubix)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=danielriddell21_rubix&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=danielriddell21_rubix)

A Rubik's cube solver in Go. The cube lives as plain data in a headless core, solved
nine different ways, with an optional 3D visualizer and a LEGO Mindstorms EV3 driver.

A port and extension of two "Coding Adventure" videos by Sebastian Lague:

- **Solving the Rubik's Cube** — https://www.youtube.com/watch?v=fy0HRViXZnE
- **I Tried Optimizing my Rubik's Cube Solver** — https://www.youtube.com/watch?v=Eysf6-E3ino

## Quick start

Requires Go 1.26+.

```sh
go run ./cmd/rubix solvers                       # list the solvers
go run ./cmd/rubix scramble -n 25 -seed 1        # a scramble + its facelet string
go run ./cmd/rubix solve --input <54-chars> --strategy multi
```

`--execute` prints the robot move/primitive plan; `verify`, `scan` and `gen-tables`
are the other subcommands.

`solve`, `replica`/`compare`, `scramble` and `solvers` take `-format json|csv` and
`-output <file>` for machine-readable results, e.g.:

```sh
go run ./cmd/rubix replica -count 8 -seed 1 -format csv -output results.csv
go run ./cmd/rubix solve --input <54-chars> --strategy multi -format json
```

## Watch it solve

The 3D visualizer is behind the `ebiten` build tag. It scrambles and solves on its
own — just run:

```sh
go run -tags ebiten ./cmd/rubix view
```

| input | action |
|-------|--------|
| drag / arrow keys | orbit |
| shift + ↑/↓, or scroll wheel | zoom |
| `space` | unfold to a flat net |
| `x` | x-ray (see all sides) |
| `r` | new scramble |
| `s` | switch solver |

See [docs/demos.md](docs/demos.md) for an animated GIF of each keybind.

<details>
<summary>Linux: OpenGL/X11 libraries</summary>

The window needs OpenGL/X11. On Debian/Ubuntu:

```sh
sudo apt install libgl1-mesa-dev libxrandr-dev libxcursor-dev libxinerama-dev libxi-dev
```
</details>

## The solvers

The nine solvers retrace the two videos' progression. Rates are measured over random
scrambles:

| # | strategy | idea | solve rate |
|---|----------|------|------------|
| 1 | `greedy`   | greedy best-first by solved-cubie count | ~10% (often stuck) |
| 2 | `sandwich` | greedy + backward-search dictionary | ~45% |
| 3 | `oriented` | orient all edges, then solve with F/B turns removed | ~33% |
| 4 | `cfop`     | Cross → F2L → OLL → PLL, staged | 100%, ~57 moves |
| 5 | `domino`   | reduce to the domino state, then solve it | 100%, ~30 moves |
| 6 | `iddfs`    | domino search via iterative deepening | 100%, ~23 moves |
| 7 | `idastar`  | + a lower-bound prune | 100%, ~23 moves |
| 8 | `prune`    | + exact prune tables (big speedup) | 100%, ~23 moves |
| 9 | `multi`    | try many reductions, keep the shortest | 100%, ~21 moves |

## Architecture

```mermaid
flowchart LR
    FS["facelet string"] --> M
    SCAN["EV3 scan<br/>(real cube)"] --> M
    SCR["scramble"] --> M
    M["cube.Cube<br/>(one shared model)"] --> SOL["solver"]
    SOL --> MV["moves"]
    MV --> PR["print"]
    MV --> EX["EV3 execute"]
    M -. live state .-> V["3D view"]
    MV -. animate .-> V
```

The visualizer and robot are optional front ends gated behind build tags, so the
default build is pure-Go:

| build | command |
|-------|---------|
| headless (default) | `go build ./...` |
| with visualizer | `go build -tags ebiten ./...` |
| EV3 brick | `CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=5 go build -tags ev3 -o rubix-ev3 ./cmd/rubix` |

## References

- Two-phase / domino reduction — Herbert Kociemba: http://kociemba.org/cube.htm
- Thistlethwaite's algorithm: https://www.jaapsch.net/puzzles/thistle.htm
- CFOP: https://jperm.net/3x3/cfop
- IDA* + pattern databases — Korf, *Finding Optimal Solutions to Rubik's Cube*:
  https://www.cs.princeton.edu/courses/archive/fall06/cos402/papers/korfrubik.pdf
- ev3dev: https://www.ev3dev.org/ · ev3go/ev3dev: https://github.com/ev3go/ev3dev
- MindCub3r: https://www.mindcuber.com/ · Ebiten: https://ebitengine.org/

Where this project deviates from the videos: [docs/limitations.md](docs/limitations.md).
