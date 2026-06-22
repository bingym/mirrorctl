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
```

Planned support for `ubuntu`, `npm`, `homebrew`, `gem`, `cargo`, and more.

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
