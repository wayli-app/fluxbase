---
title: Building from Source
description: Build the Fluxbase server binary natively on macOS and Linux. Covers system dependencies, the Go build-tag model (full / default / lite), and troubleshooting the Tesseract / Leptonica / vips CGO errors.
---

import { Card, CardGrid } from '@astrojs/starlight/components';

Build the Fluxbase server binary directly from source on macOS or Linux — no
Docker required. This is useful for local development, custom builds, or
platforms where the official Docker images don't fit.

<CardGrid stagger>
  <Card title="Just want to run it?" icon="rocket">
    Use the [official Docker image](../../deployment/docker/) or the prebuilt
    release binary — no build step needed.
  </Card>
  <Card title="Building the CLI only?" icon="terminal">
    The CLI is pure Go with no C dependencies. See
    [CLI installation → From Source](../../cli/installation/).
  </Card>
</CardGrid>

## How Fluxbase builds

Two media features are backed by **C libraries** and are compiled in via Go
[build tags](https://pkg.go.dev/cmd/go#hdr-Build_constraints):

| Feature | Build tag | C dependency | Go binding |
| --- | --- | --- | --- |
| OCR (scanned PDFs / images in knowledge bases) | `ocr` | Tesseract + Leptonica | `gosseract` |
| Image transformations (resize / reformat / rotate) | `vips` | libvips | `govips` |

Each feature has a **stub** that compiles in when its tag is absent. The stub
self-disables the feature at runtime — the server starts normally and reports
the capability as unavailable. So Fluxbase **always builds and runs**, with or
without the C libraries; the tags only decide which media features work.

> **CGO is always required**, regardless of which tags you pick. Fluxbase links
> `pg_query_go` (a C-based SQL parser), so you cannot build with
> `CGO_ENABLED=0`. A C toolchain (clang/gcc) must be present on every platform.

This gives you three build modes:

| `make` target | Tags | Needs C media libs | Result |
| --- | --- | --- | --- |
| `make build` (default) | `ocr` | Tesseract + Leptonica | OCR works; image transforms unavailable |
| `make build-full` | `ocr vips` | Tesseract + Leptonica + libvips | OCR **and** image transforms work (matches the full Docker image) |
| `make build-lite` | _(none)_ | _none_ | No C media deps; OCR and transforms self-disable (matches the lite Docker image) |

## Prerequisites

Install these regardless of which build mode you choose:

| Tool | Version | Why | Install |
| --- | --- | --- | --- |
| [Go](https://go.dev/dl/) | 1.25+ | Build the server and CLI | `brew install go` / [downloads](https://go.dev/dl/) |
| [Bun](https://bun.sh/) (or Node.js 20+) | latest | Admin UI + embedded SDKs | `brew install bun` |
| [Deno](https://deno.land/) | latest | Edge functions runtime | `brew install deno` |
| [PostgreSQL](https://www.postgresql.org/) | 16+ with [pgvector](https://github.com/pgvector/pgvector) | Target database | `brew install postgresql@16` |
| C toolchain | clang / gcc | CGO (`pg_query_go`) | included with Xcode Command Line Tools / `build-essential` |
| `pkg-config` | any | CGO locates the C libraries | `brew install pkg-config` |

> The server builds the TypeScript and React SDKs and the admin UI as part of
> `make build`. If you only need the server binary and want to skip that, run
> `go build ./cmd/fluxbase` directly after at least one `make build` (or run
> `make clean` to install the placeholder admin UI).

## System dependencies (C libraries)

Only needed for the build tags you actually enable.

| Platform | For `ocr` (Tesseract + Leptonica) | Add `vips` for image transforms | Runtime extras |
| --- | --- | --- | --- |
| **macOS** (Homebrew) | `brew install tesseract leptonica pkg-config` | add `vips` | `brew install poppler` (PDF OCR), `brew install tesseract-lang` (non-English) |
| **Debian / Ubuntu** | `sudo apt-get install -y libtesseract-dev libleptonica-dev pkg-config` | add `libvips-dev` | `sudo apt-get install -y poppler-utils tesseract-ocr tesseract-ocr-eng` |
| **Fedora / RHEL** | `sudo dnf install -y tesseract-devel leptonica-devel pkgconfig` | add `vips-devel` | `sudo dnf install -y poppler-utils tesseract tesseract-langpack-eng` |
| **Arch Linux** | `sudo pacman -S tesseract leptonica pkgconf` | add `libvips` | `sudo pacman -S poppler tesseract-data-eng` |

### About the macOS Homebrew quirk

This is the root cause of the most common build error.

Homebrew installs **leptonica** and **vips** _keg-only_ — their `pkgconfig`
files are **not** symlinked into the default search path. So on a fresh
terminal, `pkg-config` (and therefore CGO) cannot find them, and `make build`
fails with:

```
# github.com/otiai10/gosseract/v2
tessbridge.cpp:5:10: fatal error: 'leptonica/allheaders.h' file not found
```

The Fluxbase **Makefile handles this automatically**. On macOS it detects
Homebrew and wires up _both_ ways the bindings discover the libraries:

- It prepends the keg `pkgconfig` directories to `PKG_CONFIG_PATH`. This is
  what **`govips`** (the `vips` tag) needs, since it reads pkg-config.
- It also sets `CGO_CPPFLAGS` / `CGO_CXXFLAGS` / `CGO_LDFLAGS` to the keg
  include and library directories. This is required for **`gosseract`** (the
  `ocr` tag), which does **not** use pkg-config — it hardcodes the
  Intel-Homebrew paths (`/usr/local/...`), and on Apple Silicon the kegs live
  under `/opt/homebrew/opt/...` instead.

As long as you build via `make build`, you do **not** need to set any
environment variables manually, and opening a new terminal won't break the
build.

If you invoke `go build` directly (bypassing the Makefile), set the paths
yourself — see [Manual environment setup](#manual-environment-setup-macos)
below.

## Build

### 1. macOS quick path

```bash {5}
# 1. Install build tools
brew install go bun deno postgresql@16 pkg-config

# 2. Install C libraries for the default OCR build
brew install tesseract leptonica

# 3. Clone and build
git clone https://github.com/nimbleflux/fluxbase.git
cd fluxbase
make build

# 4. Run
./build/fluxbase-server
```

That's it — the Makefile takes care of the Homebrew keg paths, builds the
SDKs and admin UI, and produces `build/fluxbase-server`.

### 2. Pick a build mode

```bash
# Default: OCR enabled, no image transforms
make build

# Full: OCR + libvips image transforms (matches the full Docker image)
#       — also `brew install vips` first
make build-full

# Lite: no C media dependencies at all (matches the lite Docker image)
make build-lite

# Override tags directly if you need something custom
make build BUILD_TAGS="ocr vips"
```

### 3. Linux quick path (Debian / Ubuntu)

```bash
sudo apt-get update
sudo apt-get install -y \
  build-essential pkg-config \
  libtesseract-dev libleptonica-dev libvips-dev \
  poppler-utils tesseract-ocr tesseract-ocr-eng

git clone https://github.com/nimbleflux/fluxbase.git
cd fluxbase
make build-full
```

## Manual environment setup (macOS)

Only needed if you call `go build` directly instead of `make build` (the
Makefile does all of this for you automatically). The two media bindings
discover their libraries differently, so you may need both kinds of setup:

**For OCR** (`gosseract`, the `ocr` tag) — gosseract does **not** use
pkg-config; it hardcodes the Intel-Homebrew paths, so `PKG_CONFIG_PATH` has no
effect on it. Point CGO at the keg include/library directories directly (works
on Apple Silicon and Intel because it uses `brew --prefix`):

```bash
# ~/.zshrc or ~/.bashrc
export CGO_CPPFLAGS="-I$(brew --prefix leptonica)/include -I$(brew --prefix tesseract)/include"
export CGO_CXXFLAGS="-I$(brew --prefix leptonica)/include -I$(brew --prefix tesseract)/include"
export CGO_LDFLAGS="-L$(brew --prefix leptonica)/lib -L$(brew --prefix tesseract)/lib"
```

> `gosseract` compiles a C++ bridge (`tessbridge.cpp`), so the include flags
> must be in `CGO_CPPFLAGS` / `CGO_CXXFLAGS` — not only `CGO_CFLAGS`.

**For image transforms** (`govips`, the `vips` tag) — govips **does** use
pkg-config, so add the keg `pkgconfig` directory to the search path:

```bash
export PKG_CONFIG_PATH="$(brew --prefix vips)/lib/pkgconfig:$PKG_CONFIG_PATH"
```

Verify CGO can see the libraries:

```bash
pkg-config --cflags --libs vips   # should print -I and -L flags (vips tag)
```

## Troubleshooting

### `'leptonica/allheaders.h' file not found`

The `ocr` build tag is set but Leptonica's headers aren't on the search path.

- **Building with `make build`** — make sure
  `brew install tesseract leptonica` succeeded. The Makefile sets the CGO
  include/library paths for you, so this should just work — re-run `make build`.
- **Building with `go build` directly** — `gosseract` ignores `PKG_CONFIG_PATH`,
  so you must set `CGO_CPPFLAGS` / `CGO_CXXFLAGS` / `CGO_LDFLAGS` yourself, as
  described in [Manual environment setup](#manual-environment-setup-macos).
- **Don't need OCR?** — build lite and skip the C libraries entirely:
  `make build-lite`.

### `'tesseract/api/capi.h' file not found`

Same class of problem, but for Tesseract rather than Leptonica. The fix is
identical: `brew install tesseract` (or the Linux equivalent) and ensure
`PKG_CONFIG_PATH` includes the Tesseract keg.

### `vips/Vips8.h file not found` or duplicate-symbol errors

You're building with the `vips` tag but libvips isn't installed. Either install
it (`brew install vips` / `libvips-dev`) or drop the `vips` tag — use the
default `make build` instead of `make build-full`.

### `pkg-config: not found`

Install `pkg-config` (`brew install pkg-config` / `apt-get install pkg-config`).
CGO uses it to locate the C libraries.

### OCR or transforms report "unavailable" at runtime

This means you built **lite** (or default, for transforms) — the feature's
stub compiled in. To enable:

- OCR → rebuild with the `ocr` tag: `make build` or `make build-full`.
- Image transforms → rebuild with the `vips` tag: `make build-full`.

You can check which capabilities a running server has via the knowledge-bases
capabilities endpoint (`ocr_available`) — see
[Knowledge Bases](../../guides/knowledge-bases/).

## What you lose without each feature

- **Without `ocr`** — scanned PDFs and image-only files in AI knowledge bases
  are not text-extracted. See [Knowledge Bases](../../guides/knowledge-bases/).
- **Without `vips`** — storage image transformations (resize, reformat, rotate)
  return an error. Digital-native files and OCR on text PDFs are unaffected.
  See [Image Transformations](../../guides/image-transformations/).
- **Neither tag** — the server is fully functional for everything except OCR
  ingestion and image transforms. This is exactly the [lite Docker image](../../deployment/docker/#image-variants).
