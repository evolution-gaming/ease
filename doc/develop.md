# Bootstrap project for development
## Project layout

Project is essentially a single CLI tool thus all the interesting bits are in project
root. Internal logic is hidden inside `internal/` directory. Also there are no plans to
expose any parts of the project as a reusable library.

## Requirements

- A recent Go development toolchain (see [go.mod](../go.mod) file in project root for
  minimums Go version). Installation depends on OS, on GNU/Linux can use distribution's
  package manager. Consult https://golang.org/doc/install for installation instructions
  from upstream.
- `ffmpeg` and `ffprobe` binaries built with `libvmaf` support (with built-in
  VMAF models) available on `$PATH`. CI pipeline uses static binaries from the
  [mwader/static-ffmpeg](https://github.com/wader/static-ffmpeg) Docker image,
  which includes `libvmaf`. If your Linux distribution does not ship `ffmpeg`
  with libvmaf support, then it is possible to use static ffmpeg builds. Consult
  [ffmpeg.org download page](https://ffmpeg.org/download.html) for available
  options. `ffmpeg` is used for VMAF calculations and also in tests. `ffprobe`
  is used to get video metadata and frame statistics.

## Develop

Use static code analysis and tests frequently during development process, there are handy
*actions* defined in `build.sh` script for this - make use of them.

```
./build.sh lint
./build.sh test
```

Keep unittest coverage reasonably high (>=80%) where possible.

## Build

To build a binary for host target and architecture:

```
./build build
```

Resulting binary will be placed in `out` directory relative to project root.

When binary is built - put it somewhere in your PATH for convenience or source
`setenv.sh` file which will add `out` directory to PATH.
