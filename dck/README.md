# DCK version

This directory contains the construction-kit version of nonameno-demo. The original Go sources are preserved at their original paths (revision `49dff2f5be1cb2852b0f7a431f4e1d9e9b9050b6`), with small asset accessors so both versions use the same embedded resources.

Run the original with `go run ./cmd/nonameno` and this version with `go run ./dck/cmd/nonameno` from the repository root.

The page texts and assets remain in this repository. Reusable rendering and
effects come from the published `github.com/olivierh59500/democonstructionkit`
module pinned in `go.mod`. Music is opened with `sound.Open`; DCK selects the decoder from the asset and
provides the configured stereo PCM format. The demo keeps its playback level and loop settings.

The 160-letter page animation uses `motion.GlyphPageCycle` and
`sprites.GlyphPages`. Its 20-by-8 layout, three delay maps, 2-second elastic
entrance, 1-second elastic exit, depth sorting and completion barriers are
configured by `presets.NonamenoGlyphPages`. The page strings and initial delay
choice remain here. The delay generators can also produce serpentine,
mirrored-column or spiral entrances for other fonts and grid sizes.
