# Encoder Evaluation Suite (ease)
[![ci](https://github.com/evolution-gaming/ease/actions/workflows/ci.yaml/badge.svg)](https://github.com/evolution-gaming/ease/actions/workflows/ci.yaml)

## About

Video encoder evaluation is no easy task, it requires substantial effort and combines a
lot of "stages". This tool should help lift some of the menial burden and make it a tad
little easier.

`ease` is a non-interactive command-line tool that can be used to automate encoding and
analysis "stages" of video encoder evaluation process.

Tool name is a "play on words", *EES* phonetically sounds very close to *ease* which
carries a meaning of:

> ease *verb*:
> to make or become less severe, difficult, unpleasant, painful
>
> ease *noun*:
> the state of experiencing no difficulty, effort, pain

## Dependencies

Tool depends on `ffmpeg` and `ffprobe` version >= 5.0 built with `libvmaf` support. Both
`ffmpeg` and `ffprobe` must be on `$PATH`.

If your GNU/Linux distribution does not provide recent `ffmpeg` package or does not
include `libvmaf` then there are static builds of `ffmpeg` available from
https://johnvansickle.com/ffmpeg/.

## Install

**Note:** `ease` tool only runs on GNU/Linux and macOS platforms.

Easiest approach is to download binaries from project [releases](https://github.com/evolution-gaming/ease/releases) page and after unarchiving put `ease` binary on `$PATH`.

Or if you are comfortable with [Go](https://go.dev) tooling then:

    go install github.com/evolution-gaming/ease@latest

## Documentation

For full and up-to-date usage examples and documentation of options consult
`ease` tool help with `ease -h`.

For usage documentation and examples head to [usage documentation](doc/usage.md).

There is also documentation on [how to bootstrap project](doc/develop.md) for development
purposes.

## License
Ease tool source code is licensed under [MIT license](https://choosealicense.com/licenses/mit/), see [LICENSE](LICENSE).
