# Conventions

Structure shared across the tool family — unum, fiat-lux, galapagos, pandemonium,
vivarium, toolshed, narrata, gambit, rubix — so they read like they were written
by one person. Each repo records only the conventions it follows; this repo
follows the **CLI entrypoint** and **Ebiten GUI** conventions below. unum is the
CLI reference; rubix and vivarium are the GUI references.

These cover the *shape* of an entrypoint and a GUI, not the behaviour inside them.

## CLI entrypoint

Every binary is built the same way:

- **Thin `cmd/<name>/main.go`.** It declares `var version = "dev"` (overridden at
  release via `-ldflags "-X main.version=…"`) and does nothing but
  `cli.Execute(version)`, printing the error and exiting non-zero on failure.
- **`internal/cli` owns the command tree.** `func Execute(version string) error`
  builds the root `*cobra.Command` (with `SilenceUsage`/`SilenceErrors`),
  registers subcommands, and runs it. Keep all flag wiring here, not in `main`.
- **`--version`** comes from cobra's `Version` field. Do **not** add a bespoke
  `version` subcommand — `--version` (and cobra's `-v`) is the one way.
- **`completion`** subcommand in `internal/cli/completion.go`, generating
  bash/zsh/fish/powershell from `cmd.Root()`. Copy unum's verbatim, changing only
  the binary name.

## Ebiten GUI

Graphical front-ends follow one shape:

- **`internal/gui` is the only package that imports Ebiten.** Nothing else —
  especially not `internal/cli` — pulls in `hajimehoshi/ebiten`. Rendering helpers
  may live in their own package; only the `ebiten.Game` and the window belong in
  `internal/gui`.
- **Uniform seam.** Expose `func Run(cfg Config) error` and `func Available() bool`,
  both defined in *both* builds. `Config` lives in an untagged file (no Ebiten
  types), so the CLI constructs it and calls `gui.Run(cfg)` regardless of build tags.
- **Build-tag policy.** Repos with a real non-GUI mode gate the window behind
  `//go:build ebiten` with a `//go:build !ebiten` stub whose `Run` returns
  `"built without the GUI; rebuild with -tags ebiten…"` and whose `Available()`
  returns `false`. A pure game with no headless mode keeps Ebiten ungated (it still
  exposes the seam; `Available()` returns `true`, no stub).
- **Game conventions.** A `New(...)` constructor returns a struct implementing
  `ebiten.Game` (`Update`/`Draw`/`Layout`); window setup lives in the
  `ebiten`-tagged file of `internal/gui`, and input is decomposed into focused
  helpers rather than one large `Update`.
