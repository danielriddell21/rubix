# rubix task runner — see https://just.systems
# Run `just` (or `just --list`) to see the available recipes.

# show the available recipes
default:
    @just --list

# build the headless binary
build:
    go build ./...

# build everything, including the ebiten visualizer
build-gui:
    go build -tags ebiten ./...

# run the tests
test:
    go test ./...

# run the tests, including ebiten-tagged code
test-gui:
    go test -tags ebiten ./...

# format the code
fmt:
    gofmt -w .

# vet the code
vet:
    go vet ./...

# lint (needs golangci-lint that supports this module's Go version)
lint:
    golangci-lint run

# tidy go.mod / go.sum
tidy:
    go mod tidy

# solve & compare many cubes, e.g. `just replica "-count 8 -compare"`
replica args="":
    go run ./cmd/rubix replica {{args}}

# open the self-driving visualizer (needs the ebiten build)
view:
    go run -tags ebiten ./cmd/rubix view

# record one GIF per keybind into docs/demos/ (needs the ebiten build and a display)
demos:
    mkdir -p docs/demos
    go run -tags ebiten ./cmd/rubix view --seed 1 --record docs/demos/unfold.gif       --record-keys space
    go run -tags ebiten ./cmd/rubix view --seed 1 --record docs/demos/xray.gif         --record-keys x
    go run -tags ebiten ./cmd/rubix view --seed 1 --record docs/demos/rescramble.gif   --record-keys r
    go run -tags ebiten ./cmd/rubix view --seed 1 --record docs/demos/cycle-solver.gif --record-keys s
    go run -tags ebiten ./cmd/rubix view --seed 1 --record docs/demos/move-list.gif    --record-keys m
    go run -tags ebiten ./cmd/rubix view --seed 1 --record docs/demos/orbit.gif        --record-keys left
    go run -tags ebiten ./cmd/rubix view --seed 1 --record docs/demos/zoom.gif         --record-keys shift+up
    go run -tags ebiten ./cmd/rubix view --seed 1 --record docs/demos/focus.gif        --record-keys replica,tab,tab,tab
