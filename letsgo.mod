// letsgo.mod

// The six targets the GoReleaser config built.
build (
	linux/amd64
	linux/arm64
	darwin/amd64
	darwin/arm64
	windows/amd64
	windows/arm64
)

// The same commands built a second time with the Ebiten window compiled in.

// darwin only, as the GoReleaser config published: the GUI build used to need
// cgo for Metal and Cocoa, so it could only be built on a macOS runner. That
// is no longer true — Ebiten 2.10 goes through purego and cross-compiles with
// CGO_ENABLED=0 — but widening the list is a decision about what this game
// supports rather than part of moving release tools, so it is left alone.

// The suffix keeps the two archives apart. The binary inside both is still
// `rubix`, so the command does not depend on which one you installed.
variant gui (
	build darwin/amd64 darwin/arm64
	tags ebiten
)

// The headless build's formula. The GUI build's cask is the letsgo-cask
// plugin pinned below.
brew danielriddell21/tap

// The shared GoReleaser workflow marked releases as pre-releases after
// publishing; letsgo does it while publishing, so promote.yaml still fires on
// manual promotion.
release prerelease=true

// The gui variant's archives as a cask, written to the tap with the formula.
// Settings are in .letsgo/cask.mod.
plugin tap-files letsgo-cask v0.6.2 sha256:db20ff8adf2a15b6b7c04b9795d30cdf5c0fd45ca866996074a90f3e7a6b7f8f
