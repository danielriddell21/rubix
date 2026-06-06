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
