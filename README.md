# Encoder Evaluation Suite (ease)

Tool name is a "play on words", *EES* phonetically sounds very close to *ease* which
carries a meaning of:

> ease *verb*:
> to make or become less severe, difficult, unpleasant, painful
>
> ease *noun*:
> the state of experiencing no difficulty, effort, pain

Encoder evaluation is no easy task, it requires substantial effort and combines a lot of
"stages". This tool should help lift some of the menial burden and make it a tad little
easier.

## Project layout

Project is essentially a single CLI tool thus all the interesting bits are in project
root. Internal logic is hidden inside `internal/` directory. Also there are no plans to
expose any parts of the project as a reusable library.

## Build

**Requirements**:

- Go development toolchain. Installation depends on OS, on GNU/Linux can use
  distribution's package manager. Consult https://golang.org/doc/install for
  installation instructions from upstream.
- Version 5.X of `ffmpeg` and `ffprobe` binaries (built with `libvmaf`) available in
  `$PATH`. During development static `ffmpeg/ffprobe` binaries from
  https://johnvansickle.com/ffmpeg/ are used. **Other binaries may not work** - it depends
  on specific Linux distribution and how they choose to build ffmpeg (read - which
  features are enabled). `ffmpeg` is used for VMAF calculations and also in tests.
  `ffprobe` is used to get video metadata and frame statistics.

To build a binary for host target and architecture:

```
./build build
```

Resulting binary will be placed in `out` directory relative to project root.

When binary is built - put it somewhere in your PATH for convenience or source
`setenv.sh` file which will add `out` directory to PATH.

## Usage of ease tool

**Note:** Version 5.X of `ffmpeg` and `ffprobe` binaries (built with `libvmaf`) are
required to be available in `$PATH` for video quality calculations.

For full and up-to-date usage examples and documentation of options consult
`ease` tool help with `ease -h`.

Tool consists of a number of subcommands, at this pont following subcommands are
implemented:

- `ease encode`
- `ease analyse`
- `ease bitrate`
- `ease vqmplot`

### Encoding stage

Example of a simple batch encoding stage looks like:

```
$ ease encode -plan encoding_plan.json -report run_report.json
```

Where `encoding_plan.json` defines all encoder runs - essentially an batch
encode configuration and `run_report.json` contains encoding run metadata and
per encoding results.

Full list of options are as follows (from `ease encode -h`):

>  -plan string
>
>    	Encoding plan configuration file

Mandatory option. Path to "encoding plan" configuration file.

>  -report string
>
>    	Encoding plan report file (default is stdout)

Optional path to JSON report file.

>  -vqm
>
>    	Calculate VQMs (default true)

Controls if VQMs are calculated for this run. Since VQM calculation is CPU
intensive and time consuming - it is possible to disable VQM calculation via
`-vqm=false`. Default is to run VQM calculations for all encoded files. This
stage can be time consuming for long videos and/or on weak hardware.

>  -dry-run
>
>    	Do not actually run, just do checks and validation

Will perform a "dry run" of "encoding plan". Meaning will do validation of
configuration and other checks - no actual encodings will be performed.

### Encoding plan example

Term "encoding plan" is used in this project to refer to a single event of batch
execution of encoding commands. The encoding plan is defined by a single
configuration file. At this point configuration is defined as a simple JSON
document. It is possible that some other format of configuration will be adopted
in future.

This is a simple example of encoding plan configuration in JSON:

```json
{
    "OutDir": "x264_out",
    "Inputs": [
        "./videos/clip01.mp4",
        "./videos/clip02.mp4",
    ],
    "Schemes": [
        {
            "Name": "tbr_1700k",
            "CommandTpl": [
                "ffmpeg -i %INPUT% ",
                "-c:v libx264 -an -f mp4 -g 25 -r 25 ",
                "-tune zerolatency -preset faster ",
                "-b:v 1700k -maxrate 2100k -bufsize 2100k ",
                "-y %OUTPUT%.mp4"
            ]
        },
        {
            "Name": "tbr_2000k",
            "CommandTpl": [
                "ffmpeg -i %INPUT% ",
                "-c:v libx264 -an -f mp4 -g 25 -r 25 ",
                "-tune zerolatency -preset faster ",
                "-b:v 2000k -maxrate 2500k -bufsize 2500k ",
                "-y %OUTPUT%.mp4"
            ]
        }
    ]
}
```

Result of this encoding plan execution  would be following files:

```
x264_out/clip01_tbr_1700k.mp4
x264_out/clip01_tbr_1700k.out
x264_out/clip01_tbr_2000k.mp4
x264_out/clip01_tbr_2000k.out
x264_out/clip02_tbr_1700k.mp4
x264_out/clip02_tbr_1700k.out
x264_out/clip02_tbr_2000k.mp4
x264_out/clip02_tbr_2000k.out
```

- `OutDir` is directory in which to save encoded/compressed files and log output
  generated by encoder command.
- `Inputs` is an array of source/mezzanine video files that are subject to
  compression
- `Schemes` is an array that contains various encoder commands. This is
  basically a list of all encoder command lines that are part of this encoding
  plan and will be executed for each source video defined in `Inputs`.
- Scheme `Name` is a name for specific encoder command, this of it as some
  meaningful nomenclature for this specific encoding experiment. This name will
  be used in compressed file filename - so keep it sane.
- Scheme `CommandTpl` is a "template" for executing a specific encoding
  experiment. It is basically an encoder command-line with `%INPUT%` and
  `%OUTPUT%` placeholders.

  Also worth noting that `CommandTpl` is an array of strings, reason for this is
  to have ability to split long encoder command-lines into "multi-lines" thus
  making it easier on human eyes. Elements of array are joined together later on
  into single string, so keep this in ming and put trailing spaces where needed.

### Analysis stage

To aid in analysis part of encoded videos there is `ease analyse` subcommand.
This subcommand requires artifacts from previous encoding stage. For time being
input for `ease analyse` is encoding plan report generated by `ease encode`
tool.

Example usage:

```
ease analyse -report path/to/ease-encode-generated/report.json -out-dir analysis
```

Analysis artifacts will be placed in directory specified with option `-out-dir`.
There artifacts include:

- Bitrate plot (aggregated into 1s buckets) and frame size plot
- VMAF, PSNR and MS-SSIM metrics related plots (per-frame , histogram,
  Cumulative Distribution Function)

### Other subcommands

For convenience purposes there are also 2 other subcommands - namely `bitrate`
and `vqmplot`, these will create bitrate plot for a given video file and create
VQM plot from *libvmaf* generated JSON report accordingly. Again, consult each
subcommand's help e.g. `ease bitrate -h` and `ease vqmplot -h` for full help.

Examples `bitrate` usage:

```
ease bitrate -i my_video.mpx -o by_video_bitrate.png
```

Examples `vqmplot` usage:

```
ease vqmplot -m PSNR -i libvmaf.json -o psnr.png
```
