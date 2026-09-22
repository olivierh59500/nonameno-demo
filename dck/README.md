# DCK version

This directory contains the construction-kit version of nonameno-demo. The original Go sources are preserved at their original paths (revision `49dff2f5be1cb2852b0f7a431f4e1d9e9b9050b6`), with small asset accessors so both versions use the same embedded resources.

Run the original with `go run ./cmd/nonameno` and this version with `go run ./dck/cmd/nonameno` from the repository root.

The choreography and assets remain in this repository. Reusable rendering and
effects come from the published `github.com/olivierh59500/democonstructionkit`
module pinned in `go.mod`. Go downloads the dependencies automatically, including
`github.com/olivierh59500/ym-player v1.0.0` for YM playback. Second Reality retains its original ST3 music synchronization.
