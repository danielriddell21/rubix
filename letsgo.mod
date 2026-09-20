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

// The headless build's formula. The GUI build's cask is written after the
// release by letsgo-cask, which letsgo does not run and cannot be changed by.
brew danielriddell21/tap

// The shared GoReleaser workflow marked releases as pre-releases after
// publishing; letsgo does it while publishing, so promote.yaml still fires on
// manual promotion.
release prerelease=true
