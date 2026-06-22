# mirrorctl

A command-line tool written in **Go** for switching package manager / system mirror sources with a single command.

## Usage

```
mirrorctl [type] [action] [value]
```

Examples:

```sh
mirrorctl pypi config          # Show current pip mirror configuration
mirrorctl pypi set tuna        # Switch pip to Tsinghua mirror
mirrorctl pypi set aliyun      # Switch to Alibaba Cloud mirror
mirrorctl pypi set ustc        # Switch to USTC mirror
mirrorctl pypi list            # List all available PyPI mirrors
mirrorctl pypi unset           # Remove pip mirror config, restore default

mirrorctl github convert https://github.com/user/repo          # Convert GitHub URL to proxy URL
mirrorctl github download https://github.com/user/repo/archive/refs/heads/main.zip  # Download via proxy
mirrorctl github clone user/repo                               # Clone repo via proxy accelerator

mirrorctl ubuntu config                # Show current Ubuntu APT mirror
mirrorctl ubuntu set tuna              # Switch Ubuntu to Tsinghua TUNA mirror
mirrorctl ubuntu list                  # List available Ubuntu mirrors
mirrorctl debian config                # Show current Debian APT mirror
mirrorctl debian set ustc              # Switch Debian to USTC mirror

mirrorctl npm config                   # Show current npm registry
mirrorctl npm set npmmirror            # Switch npm to npmmirror (Taobao)
mirrorctl yarn set npmmirror           # Switch Yarn (Berry) to npmmirror
mirrorctl pnpm set npmmirror           # Switch pnpm to npmmirror
mirrorctl uv set tuna                  # Switch uv to TUNA PyPI mirror
mirrorctl brew set tuna                # Switch Homebrew to TUNA mirror
```

Planned support for `gem`, `cargo`, and more.

### docker

Convert and pull Docker images through an acceleration proxy (gh-proxy.org/docker/):

```sh
mirrorctl docker convert nginx:latest                           # Convert image ref to proxy URL
mirrorctl docker pull nginx:latest                               # Pull image via proxy accelerator
mirrorctl docker pull --strip nginx:latest                       # Pull and strip proxy prefix from image name
mirrorctl docker pull gcr.io/kaniko-project/executor:debug       # Pull from GCR via proxy
```

## Building

Requirements: Go 1.21+ and make.

```sh
make            # Build ./bin/mirrorctl
make run ARGS="pypi list"   # Quick run, pass arguments via ARGS
make test       # Run unit tests
make vet        # Static analysis
make fmt        # Format code
make clean      # Clean up
```

You can also use the Go toolchain directly:

```sh
go build -o bin/mirrorctl .
go test ./...
go run . pypi list
```

## Supported Mirrors

### pypi

| name   | Maintainer                          | index-url                                  |
|--------|-------------------------------------|--------------------------------------------|
| tuna   | Tsinghua University TUNA            | https://pypi.tuna.tsinghua.edu.cn/simple/  |
| aliyun | Alibaba Cloud                       | https://mirrors.aliyun.com/pypi/simple/     |
| ustc   | University of Science and Technology of China | https://pypi.mirrors.ustc.edu.cn/simple/ |

Config file location (XDG path `~/.config/pip/pip.conf` takes priority, falls back to `~/.pip/pip.conf`):

```ini
[global]
index-url = https://pypi.tuna.tsinghua.edu.cn/simple/

[install]
trusted-host = pypi.tuna.tsinghua.edu.cn
```

### goproxy

| name       | Maintainer          | URL                    |
|------------|---------------------|------------------------|
| goproxy.io | Global Go proxy     | https://goproxy.io     |
| goproxy.cn | Qiniu (China)       | https://goproxy.cn     |

Uses `go env -w GOPROXY=<url>,direct` to set the proxy. Configuration is stored in `$GOENV` (typically `~/.config/go/env`).

### github

| Feature | Proxy URL                          | Description                              |
|---------|------------------------------------|------------------------------------------|
| convert | `https://gh-proxy.com/...+URL`     | Convert GitHub URL to proxy URL          |
| download| curl via proxy                     | Download files through proxy accelerator |
| clone   | git clone via proxy                | Clone repos through proxy accelerator    |

The proxy prefix `https://gh-proxy.com/` is prepended to GitHub URLs to route traffic through CDN-accelerated nodes, improving access speed in regions where GitHub is slow or unreliable.

### docker

| Feature | Proxy URL                              | Description                              |
|---------|----------------------------------------|------------------------------------------|
| convert | `gh-proxy.org/docker/...+ref`          | Convert Docker image ref to proxy ref    |
| pull    | docker pull via proxy                  | Pull Docker images through proxy accelerator |

The proxy prefix `gh-proxy.org/docker/` is prepended to image references (used as a Docker registry host/path). Supports all public registries including Docker Hub, GCR, GHCR, Quay, MCR, and more.

### ubuntu

| name     | Maintainer                          | URL                                          |
|----------|-------------------------------------|----------------------------------------------|
| ustc     | USTC                                | https://mirrors.ustc.edu.cn/ubuntu/          |
| tuna     | Tsinghua University TUNA            | https://mirrors.tuna.tsinghua.edu.cn/ubuntu/ |
| aliyun   | Alibaba Cloud                       | https://mirrors.aliyun.com/ubuntu/           |
| 163      | NetEase 163                         | https://mirrors.163.com/ubuntu/              |
| huawei   | Huawei Cloud                        | https://mirrors.huaweicloud.com/ubuntu/      |
| tencent  | Tencent Cloud                       | https://mirrors.tencent.com/ubuntu/          |

Replaces `archive.ubuntu.com` and `security.ubuntu.com` with the mirror hostname in `/etc/apt/sources.list` or `/etc/apt/sources.list.d/ubuntu.sources` (auto-detects DEB822 vs traditional format). Requires root.

### debian

| name     | Maintainer                          | URL                                          |
|----------|-------------------------------------|----------------------------------------------|
| ustc     | USTC                                | https://mirrors.ustc.edu.cn/debian/          |
| tuna     | Tsinghua University TUNA            | https://mirrors.tuna.tsinghua.edu.cn/debian/ |
| aliyun   | Alibaba Cloud                       | https://mirrors.aliyun.com/debian/           |
| 163      | NetEase 163                         | https://mirrors.163.com/debian/              |
| huawei   | Huawei Cloud                        | https://mirrors.huaweicloud.com/debian/      |

Replaces `deb.debian.org` with the mirror hostname in `/etc/apt/sources.list` or `/etc/apt/sources.list.d/debian.sources` (auto-detects DEB822 vs traditional format). Requires root.

### npm / yarn / pnpm

| name     | Maintainer                          | registry URL                                     |
|----------|-------------------------------------|--------------------------------------------------|
| npmmirror| npmmirror (formerly Taobao)          | https://registry.npmmirror.com/                  |
| tuna     | Tsinghua University TUNA            | https://mirrors.tuna.tsinghua.edu.cn/npm/        |
| huawei   | Huawei Cloud                        | https://repo.huaweicloud.com/repository/npm/     |
| tencent  | Tencent Cloud                       | https://mirrors.tencent.com/npm/                 |

npm writes `registry=<url>` to `~/.npmrc`. yarn writes `npmRegistryServer: "<url>"` to `~/.yarnrc.yml`. pnpm uses `pnpm config set registry <url>`.

### uv

| name   | Maintainer                          | index URL                                       | python-install-mirror                        |
|--------|-------------------------------------|-------------------------------------------------|----------------------------------------------|
| tuna   | Tsinghua University TUNA            | https://pypi.tuna.tsinghua.edu.cn/simple/       | https://mirrors.tuna.tsinghua.edu.cn/python/ |
| aliyun | Alibaba Cloud                       | https://mirrors.aliyun.com/pypi/simple/         | https://mirrors.aliyun.com/python/           |
| ustc   | USTC                                | https://pypi.mirrors.ustc.edu.cn/simple/        | https://mirrors.ustc.edu.cn/python/          |

Writes `[[index]]` + `url` for `uv pip install` and `python-install-mirror` for `uv python install` to `~/.config/uv/uv.toml`.

### brew

| name   | Maintainer               | brew git remote                                           | core git remote                                              | bottle domain                                      |
|--------|--------------------------|-----------------------------------------------------------|--------------------------------------------------------------|----------------------------------------------------|
| tuna   | Tsinghua University TUNA | https://mirrors.tuna.tsinghua.edu.cn/git/homebrew/brew.git | https://mirrors.tuna.tsinghua.edu.cn/git/homebrew/homebrew-core.git | https://mirrors.tuna.tsinghua.edu.cn/homebrew-bottles |
| ustc   | USTC                     | https://mirrors.ustc.edu.cn/brew.git                      | https://mirrors.ustc.edu.cn/homebrew-core.git                | https://mirrors.ustc.edu.cn/homebrew-bottles       |
| huawei | Huawei Cloud             | https://repo.huaweicloud.com/homebrew/brew.git            | https://repo.huaweicloud.com/homebrew/homebrew-core.git      | https://repo.huaweicloud.com/homebrew-bottles      |

Sets git remote URLs for `Homebrew/brew` and `Homebrew/homebrew-core`, and writes `HOMEBREW_BOTTLE_DOMAIN` to `~/.config/mirrorctl/brew.env`. Requires `brew` installed.

## Project Structure

```
mirrorctl/
├── README.md
├── AGENTS.md
├── Makefile
├── go.mod                  # Go module definition
├── main.go                 # Entry point + CLI dispatch
├── internal/
│   ├── util/
│   │   ├── util.go         # Common utilities: path expansion, file I/O, error reporting
│   │   └── util_test.go
│   ├── pypi/
│   │   ├── pypi.go         # pypi subcommand implementation
│   │   └── pypi_test.go
	│   └── goproxy/
│       ├── goproxy.go      # goproxy subcommand implementation
│       └── goproxy_test.go
│   ├── github/
│   │   └── github.go       # github subcommand implementation (convert/download/clone)
│   ├── docker/
│   │   └── docker.go       # docker subcommand implementation (convert/pull)
│   ├── apt/
│   │   └── apt.go          # Shared Debian/Ubuntu apt source management
│   ├── ubuntu/
│   │   └── ubuntu.go       # ubuntu subcommand implementation (config/set/list/unset)
│   ├── debian/
│   │   └── debian.go       # debian subcommand implementation (config/set/list/unset)
│   ├── npm/
│   │   └── npm.go          # npm subcommand implementation (config/set/list/unset/test)
│   ├── yarn/
│   │   └── yarn.go         # yarn subcommand implementation (config/set/list/unset/test)
│   ├── pnpm/
│   │   └── pnpm.go         # pnpm subcommand implementation (config/set/list/unset/test)
│   ├── uv/
│   │   └── uv.go           # uv subcommand implementation (config/set/list/unset/test)
│   └── brew/
│       └── brew.go         # brew subcommand implementation (config/set/list/unset/test)
└── docs/
```

## TODO

### Infrastructure
- [x] Project skeleton (README / AGENTS / Makefile / go.mod)
- [x] CLI argument parsing and subcommand dispatch with cobra (main.go)
- [x] Common utility package util (path expansion, file I/O, error reporting)
- [x] PyPI mirror source data table
- [x] Go module proxy mirror source data table
- [x] Go unit tests
- [x] GitHub proxy accelerator (convert / download / clone)
- [x] Docker image proxy accelerator (convert / pull)
- [x] Ubuntu APT mirror support (config / set / list / unset)
- [x] Debian APT mirror support (config / set / list / unset)
- [x] npm registry mirror support (config / set / list / unset / test)
- [x] yarn registry mirror support (config / set / list / unset / test)
- [x] pnpm registry mirror support (config / set / list / unset / test)
- [x] uv index mirror support (config / set / list / unset / test)
- [x] Homebrew mirror support (config / set / list / unset / test)

### Features

- [x] Detect latency for each mirror (pypi + goproxy)

### pypi (pip)
- [x] `pypi config` — Read and display current pip mirror configuration
- [x] `pypi set <name>` — Write pip.conf to switch mirrors
- [x] `pypi list` — List all supported mirrors
- [x] `pypi unset` — Remove mirror config, restore default
- [x] `pypi test` — Test connectivity and latency of all PyPI mirrors (ms)

### goproxy (Go modules)
- [x] `goproxy config` — Show current GOPROXY setting
- [x] `goproxy set <name>` — Set GOPROXY via `go env -w`
- [x] `goproxy list` — List all supported Go module proxy mirrors
- [x] `goproxy unset` — Remove GOPROXY setting via `go env -u`
- [x] `goproxy test` — Test connectivity and latency of all Go module proxy mirrors

### github (GitHub proxy accelerator)
- [x] `github convert <url>` — Convert GitHub URL to proxy URL
- [x] `github download <url>` — Download files via proxy (curl)
- [x] `github clone <url>` — Clone repos via proxy (git clone)

### docker (Docker image proxy accelerator)
- [x] `docker convert <image>` — Convert Docker image ref to proxy URL
- [x] `docker pull <image>` — Pull Docker images via proxy

### ubuntu (APT)
- [x] `ubuntu config` — Show current Ubuntu APT mirror configuration
- [x] `ubuntu set <name>` — Rewrite apt sources (DEB822 + traditional formats)
- [x] `ubuntu list` — List all supported Ubuntu mirrors
- [x] `ubuntu unset` — Restore from backup

### debian (APT)
- [x] `debian config` — Show current Debian APT mirror configuration
- [x] `debian set <name>` — Rewrite apt sources (DEB822 + traditional formats)
- [x] `debian list` — List all supported Debian mirrors
- [x] `debian unset` — Restore from backup

### npm
- [x] `npm config` — Show current npm registry configuration
- [x] `npm set <name>` — Set npm registry via ~/.npmrc
- [x] `npm list` — List all supported npm mirrors
- [x] `npm unset` — Restore from backup
- [x] `npm test` — Test connectivity and latency of all npm mirrors

### yarn
- [x] `yarn config` — Show current Yarn registry configuration
- [x] `yarn set <name>` — Set Yarn registry via ~/.yarnrc.yml
- [x] `yarn list` — List all supported Yarn mirrors
- [x] `yarn unset` — Restore from backup
- [x] `yarn test` — Test connectivity and latency of all Yarn mirrors

### pnpm
- [x] `pnpm config` — Show current pnpm registry configuration
- [x] `pnpm set <name>` — Set pnpm registry via `pnpm config set`
- [x] `pnpm list` — List all supported pnpm mirrors
- [x] `pnpm unset` — Restore default via `pnpm config delete`
- [x] `pnpm test` — Test connectivity and latency of all pnpm mirrors

### uv
- [x] `uv config` — Show current uv index configuration
- [x] `uv set <name>` — Set uv index + python-install-mirror via ~/.config/uv/uv.toml
- [x] `uv list` — List all supported uv mirrors
- [x] `uv unset` — Restore from backup
- [x] `uv test` — Test connectivity and latency of all uv mirrors

### brew
- [x] `brew config` — Show current Homebrew mirror configuration (git remotes + env file)
- [x] `brew set <name>` — Set Homebrew git remotes + bottle domain
- [x] `brew list` — List all supported brew mirrors
- [x] `brew unset` — Restore git remotes to GitHub upstream
- [x] `brew test` — Test connectivity and latency of all brew mirrors

### Other types (planned)
- [ ] `gem set <name>` — Change `gem` source
- [ ] `cargo set <name>` — Write `~/.cargo/config.toml`

### Engineering
- [x] Unit tests (Go standard testing package)
- [x] Migrate CLI to cobra (auto help, completion, `--version`)
- [ ] CI (GitHub Actions, linux + macos)
- [ ] `--dry-run` / `-v` and other global options
- [ ] Colored output

## License

MIT
