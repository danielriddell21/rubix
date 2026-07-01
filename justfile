# rubix task runner — see https://just.systems
# Run `just` (or `just --list`) to see the available recipes.

# show the available recipes
default:
    @just --list

# build the headless binary
[group('build')]
build:
    go build ./...

# run the tests
[group('test')]
test:
    go test ./...

# run the linter
[group('dev')]
lint:
    golangci-lint run

# format the code
[group('dev')]
fmt:
    golangci-lint fmt

# tidy module dependencies
[group('dev')]
tidy:
    go mod tidy

# full gate: lint + test + build. all must pass before committing
[group('dev')]
ci: lint test build

# build everything, including the ebiten visualizer
[group('build')]
build-gui:
    go build -tags ebiten ./...

# run the tests, including ebiten-tagged code
[group('test')]
test-gui:
    go test -tags ebiten ./...

# solve & compare many cubes, e.g. `just replica "--count 8 --compare"`
[group('run')]
replica args="":
    go run ./cmd/rubix replica {{args}}

# open the self-driving visualizer (needs the ebiten build)
[group('run')]
view:
    go run -tags ebiten ./cmd/rubix view

# record one GIF per keybind into docs/demos/ (needs the ebiten build and a display)
[group('run')]
demos:
    mkdir -p docs/demos
    go run -tags ebiten ./cmd/rubix view --seed 1 --record docs/demos/unfold.gif       --record-keys space
    go run -tags ebiten ./cmd/rubix view --seed 1 --record docs/demos/xray.gif         --record-keys x
    go run -tags ebiten ./cmd/rubix view --seed 1 --record docs/demos/rescramble.gif   --record-keys r
    go run -tags ebiten ./cmd/rubix view --seed 1 --record docs/demos/cycle-solver.gif --record-keys s
    go run -tags ebiten ./cmd/rubix view --seed 1 --record docs/demos/move-list.gif    --record-keys m
    go run -tags ebiten ./cmd/rubix view --seed 1 --record docs/demos/orbit.gif        --record-keys left
    go run -tags ebiten ./cmd/rubix view --seed 1 --record docs/demos/zoom.gif         --record-keys shift+up
    go run -tags ebiten ./cmd/rubix view --seed 1 --record docs/demos/compare.gif      --record-keys plus,plus,plus,tab,tab,tab
