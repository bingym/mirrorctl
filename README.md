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
```

Planned support for `ubuntu`, `npm`, `homebrew`, `gem`, `cargo`, and more.

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
│   └── github/
│       └── github.go       # github subcommand implementation (convert/download/clone)
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

### Other types (planned)
- [ ] `ubuntu set <name>` — Rewrite `/etc/apt/sources.list`
- [ ] `npm set <name>` — Write `.npmrc`
- [ ] `homebrew set <name>` — Set `HOMEBREW_BOTTLE_DOMAIN` and other env vars
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
