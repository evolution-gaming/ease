# Usage of ease tool

**Note:** Version 5.X of `ffmpeg` and `ffprobe` binaries (built with `libvmaf`) are
required to be available on `$PATH` for video quality calculations.

For full and up-to-date usage examples and documentation of options consult
`ease` tool help with `ease -h`.

Tool consists of a number of subcommands, at this point following subcommands are
implemented:

- `ease run`
- `ease bitrate`
- `ease vqmplot`
- `ease dump-conf`
- `ease version`

## Intended usage workflow

Usual workflow consists of a number of logical stages:

- Preparation of [encoding plan](#encoding-plan): this defines what to encoded (your
  mezzanine clips) and how to encode (encoder string/scheme/command-line). Think of this
  as sort of declarative approach to defining batch-encode commands.

- [Running encoding plan](#run-encoding-plan): based on what is defined in encoding plan -
  perform encoding according to defined encoding scheme, save compressed outputs, run VMAF
  calculations, save results and calculate VQ metrics (VMAF, PSNR and MS-SSIM) and create
  metrics plots (per-frame , histogram, Cumulative Distribution Function)

## Run encoding plan

A sample command to run encodings according to given plan:

```
$ ease run -plan encoding_plan.json -out-dir results
```

Where `encoding_plan.json` defines all encoder runs - essentially an batch encode
configuration and `results` is a directory where all by-products of encoding will be
saved.

Full list of options are as follows (from `ease run -h`):

>  -plan string
>
>    	Encoding plan configuration file

Mandatory option. Path to "encoding plan" configuration file.

>  -out-dir string
>
>    	Output directory to store results

Mandatory option. Path to directory fo saving results. It will include plan execution
report `report.json`, actual compressed clips, encoder logs, VQM logs, VQM plots etc.

>  -dry-run
>
>    	Do not actually run, just do checks and validation

Will perform a "dry run" of "encoding plan". Meaning will do validation of
configuration and other checks - no actual encodings will be performed.

>  -conf string
>     Application configuration file path (optional)

User can specify path to ease configuration file.

## Encoding plan

Term "encoding plan" is used in this project to refer to a single event of batch
execution of encoding commands. The encoding plan is defined by a single
configuration file. At this point configuration is defined as a simple JSON
document. It is possible that some other format of configuration will be adopted
in future.

This is a simple example of encoding plan configuration in JSON:

```json
{
    "Inputs": [
        "./videos/clip01.mp4",
        "./videos/clip02.mp4"
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

JSON configuration file consists of following:

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

If we would execute this sample encoding plan with `ease` tool via:

```
$ ease run -plan encoding_plan.json -out-dir out
```

As a result we would get following file structure:

```
out/
├── clip01_tbr_1700k
│   ├── clip01_tbr_1700k_bitrate.png
│   ├── clip01_tbr_1700k_ms-ssim.png
│   ├── clip01_tbr_1700k_psnr.png
│   └── clip01_tbr_1700k_vmaf.png
├── clip01_tbr_1700k.mp4
├── clip01_tbr_1700k.out
├── clip01_tbr_1700k_vqm.json
├── clip01_tbr_2000k
│   ├── clip01_tbr_2000k_bitrate.png
│   ├── clip01_tbr_2000k_ms-ssim.png
│   ├── clip01_tbr_2000k_psnr.png
│   └── clip01_tbr_2000k_vmaf.png
├── clip01_tbr_2000k.mp4
├── clip01_tbr_2000k.out
├── clip01_tbr_2000k_vqm.json
├── clip02_tbr_1700k
│   ├── clip02_tbr_1700k_bitrate.png
│   ├── clip02_tbr_1700k_ms-ssim.png
│   ├── clip02_tbr_1700k_psnr.png
│   └── clip02_tbr_1700k_vmaf.png
├── clip02_tbr_1700k.mp4
├── clip02_tbr_1700k.out
├── clip02_tbr_1700k_vqm.json
├── clip02_tbr_2000k
│   ├── clip02_tbr_2000k_bitrate.png
│   ├── clip02_tbr_2000k_ms-ssim.png
│   ├── clip02_tbr_2000k_psnr.png
│   └── clip02_tbr_2000k_vmaf.png
├── clip02_tbr_2000k.mp4
├── clip02_tbr_2000k.out
├── clip02_tbr_2000k_vqm.json
└── report.json
```

Compressed clips `*.mp4` along with encoder generated log output `*.out` and libvmaf log
output `*.json` are saved into `out/` directory as specified by `-out-dir` commandline
flag. Encoding run result is saved to `report.json`. Also, for each compressed file there
is a subdirectory containing VQM plots `*.png` files.

## Other subcommands

For convenience purposes there are also few other subcommands - namely `bitrate`,
`vqmplot`, `dump-conf` and `version`.

`bitrate` and `vqmplot` will create bitrate plot for a given video file and create VQM
plot from *libvmaf* generated JSON report accordingly. Again, consult each subcommand's
help e.g. `ease bitrate -h` and `ease vqmplot -h` for full help.

`version` will print `ease` tool version.

`dump-conf` will print application configuration in JSON format. This configuration can be
overridden via a configuration file and can be used via `-conf` flag for most subcommands.

Examples `bitrate` usage:

```
ease bitrate -i my_video.mpx -o by_video_bitrate.png
```

Examples `vqmplot` usage:

```
ease vqmplot -m PSNR -i libvmaf.json -o psnr.png
```

## Configuration override

For certain scenarios, you might find it necessary to adjust the internal settings, such
as specifying custom paths for `ffmpeg` and `ffprobe`, or tweaking options for the
`libvmaf` filter. These adjustments can be seamlessly accomplished using a JSON
configuration file. The `ease` tool supports the `-conf` command-line flag, allowing you
to provide the configuration file's location. To familiarize yourself with the
configurable options, you can generate a template of the default configuration by
executing `ease dump-conf`:

```
$ ease dump-conf
{
  "ffmpeg_path": "/usr/bin/ffmpeg",
  "ffprobe_path": "/usr/bin/ffprobe",
  "libvmaf_model_path": "/usr/share/model/vmaf_v0.6.1.json",
  "ffmpeg_vmaf_template": "-hide_banner -i {{.CompressedFile}} -i {{.SourceFile}} -lavfi libvmaf=n_subsample=1:log_path={{.ResultFile}}:feature=name=psnr:log_fmt=json:model=path={{.ModelPath}}:n_threads={{.NThreads}} -f null -",
  "report_file_name": "report.json"
}
```

Alternatively you can also dump default configuration to file for modification:

```
$ ease dump-conf > my_config.json
```

To customize your setup, create a configuration file adjusting any or all of these
options. Then, when using the `ease` tool, simply reference your configuration file's path
with the `-conf` flag.

```
$ ease run -plan encoding_plan.json -out-dir out -conf <path/to/config.json>
```
