# v2dat

![image](https://img.shields.io/github/downloads/DanielLavrushin/v2dat/total?label=total%20downloads)

A fast, no‑frills CLI for unpacking Xray‑core / V2Ray data packs – the geoip.dat and geosite.dat – into plain‑text files.

## Features

- Unpack `geoip.dat` & `geosite.dat` in one command.
- Stream mode – reads only the entries you ask for; huge speed‑ups when filtering.
- `-t`, `--tags` flag prints every tag present in the file and exits.
- `-p`, `--print` dumps straight to stdout.
- Consistent output naming: `<dat_filename>_<tag>.txt`.
- Pre‑built binaries for Linux, macOS, Windows.

## Usage

```shell
v2dat unpack geoip [-o output_dir] [-p] [-f tag]... geoip_file
v2dat unpack geosite [-o output_dir] [-p] [-f tag[@attr]...]... geosite.dat
```

- If `-o` was omitted, the current working dir `.` will be used.
- If no filter `-f` was given. All tags will be unpacked.
- If multiple `@attr` were given. Entries that don't contain any of given attrs will be ignored.
- Unpacked text files will be named as `<geo_filename>_<filter>.txt`.
- Use `-p` instead of `-o` for stdout.

## Example

```bash
# unpack every CIDR list inside geoip.dat into ./out
v2dat unpack geoip -o out /path/to/geoip.dat

# print Danish (dansk) CIDRs to stdout
v2dat unpack geoip -p -f dk /path/to/geoip.dat

# list available tags inside geosite
v2dat unpack geosite -t /path/to/geosite.dat

# unpack only the "google" domain rules under the "cn" tag

# a) list availble attirbutes
   v2dat unpack geosite -t /path/to/geosite.dat

# b) unpack google category and stdout
   v2dat unpack geosite -o out -f google /path/to/geosite.dat
```
