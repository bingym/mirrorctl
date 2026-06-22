# AGENTS.md

This file is for coding agents (and human collaborators) to quickly understand the current state and conventions of the `mirrorctl` project. **Read this file before modifying code.**

## One-Line Introduction

`mirrorctl` is a command-line tool written in **Go** for switching package manager / system mirror sources with a single command. Usage:

```
mirrorctl [type] [action] [value]
```

For example, `mirrorctl pypi set tuna`. Code style preferences: **clear, modular, idiomatic Go**, readability first.

## Current Progress

See the TODO section in `README.md` for details. Brief status:

- ✅ CLI skeleton with [cobra](https://github.com/spf13/cobra) (`main.go` root command)
- ✅ pypi subcommand: `config` / `set <name>` / `list` / `unset` / `test`
- ✅ Unit tests (Go standard `testing` package)
- 🔜 Next: `ubuntu set <name>`
- ❌ Not yet implemented: npm / homebrew / gem / cargo / go types

## Directory Structure

```
mirrorctl/
├── README.md            # User docs + TODO (keep in sync)
├── AGENTS.md            # This file
├── Makefile             # Build system (make / go build)
├── go.mod               # Go module definition
├── main.go              # Entry point: cobra root command, registers type commands
├── docs/                # Design notes, mirror reference tables, etc.
└── internal/
    ├── util/
    │   ├── util.go      # Common utilities: error reporting, path expansion, file I/O, atomic write
    │   └── util_test.go
    └── pypi/
        ├── pypi.go      # pypi subcommand implementation (mirror table + four actions)
        └── pypi_test.go
```

### File Responsibilities

| File                   | Responsibility                                                          |
|------------------------|------------------------------------------------------------------------|
| `main.go`              | `main()` entry point; defines the cobra root command and registers type subcommands via `rootCmd.AddCommand()`. Only dispatches, no concrete logic. |
| `internal/util/util.go`| Common utilities: `Die()` for fatal errors, `ExpandHome()` for `~` expansion, `ReadFile()` / `WriteFileAtomic()` and other general-purpose functions. |
| `internal/pypi/pypi.go`| Implements pypi subcommand (config / set / list / unset / test). Maintains `Mirrors` slice, knows pip config file paths. Exposes `Command() *cobra.Command`. |

## Design Conventions

### 1. Subcommand Dispatch Pattern (cobra)

The CLI uses [cobra](https://github.com/spf13/cobra) for command parsing. Each `type` (pypi / ubuntu / npm ...) is an independent package under `internal/`, exposing a function:

```go
func Command() *cobra.Command
```

`main.go` defines the root command and registers type commands:

```go
var rootCmd = &cobra.Command{
    Use:   "mirrorctl",
    Short: "Switch package manager mirror sources",
}

func init() {
    rootCmd.AddCommand(pypi.Command())
    // future: rootCmd.AddCommand(ubuntu.Command())
}
```

Each type package builds its own cobra command tree with subcommands (config / list / set / unset). The internal `cmdConfig()`, `cmdList()`, `cmdSet()`, `cmdUnset()` functions still return `int` (0 = success, non-zero = failure) and call `os.Exit()` from the cobra `Run` handler.

**Effort to add a new type**: Create `internal/<type>/<type>.go`, implement `Command() *cobra.Command`, call `rootCmd.AddCommand(<type>.Command())` in `main.go`, update README's TODO.

### 2. Mirror Source Data Table

Each type package maintains a static table (slice of struct) internally:

```go
type Mirror struct {
    Name        string // short name for user input, e.g. "tuna"
    URL         string // full index-url
    TrustedHost string // hostname for trusted-host (may be empty)
    Desc        string // human-readable description, used by list subcommand
}

var Mirrors = []Mirror{ ... }
```

Adding a new mirror only requires adding one line to the table. **Do not write if-else chains in command handler logic.**

### 3. Error Handling

- Fatal errors (file cannot be opened, etc.): Call `util.Die(fmt, ...)` or `util.DieErr(what, err)`, prints to stderr and `os.Exit(1)`.
- User operation errors (e.g. `set <unknown name>`): Print a clear message to stderr, return non-zero exit code.
- Command handlers return `0` for success, non-zero for failure (POSIX convention). Cobra's `Run` handler calls `os.Exit()` with the return code.

### 4. Paths and File I/O

- `~` expansion: Do not hand-roll it; use `util.ExpandHome()`, which falls back to `os/user.Current()` when `$HOME` is not set.
- Before writing config files, `os.MkdirAll` the parent directory (`util.MkdirAll()`).
- File writing uses "write temp file + rename" pattern for atomic replacement (see `util.WriteFileAtomic()`).
- File reading uses `util.ReadFile()`, which returns `nil, nil` when the file does not exist.

### 5. Coding Standards

- Follow `gofmt` standard formatting.
- Package names use lowercase words, no underscores (`internal/pypi`, not `internal/py_pi`).
- Public functions/types use PascalCase (`Dispatch`, `Mirror`), private ones use camelCase (`cmdSet`, `findMirror`).
- Use `go vet` and `go test` to ensure code quality.
- String concatenation prefers `strings.Builder` + `fmt.Fprintf`.
- The only third-party dependency is `github.com/spf13/cobra` (and its transitive deps `pflag`, `mousetrap`).

### 6. Build System

Uses `Makefile` to wrap `go build` / `go test`, making it accessible to users unfamiliar with Go. Targets:

- `make` — Build `./bin/mirrorctl`
- `make clean` — Remove build artifacts
- `make run ARGS="..."` — Quick run
- `make test` — Run `go test ./...`
- `make vet` — Run `go vet ./...`
- `make fmt` — Run `go fmt ./...`

### 7. Testing Conventions

- Test files are in the same directory as source files, named `*_test.go` (Go convention).
- Use standard `testing` package; do not introduce testify or other third-party libraries.
- File system operations are performed in `t.TempDir()` to avoid polluting the real environment.
- Run tests: `go test ./...` or `make test`.

## How to Add a New Type (using `ubuntu` as example)

1. Check off the corresponding item in `README.md` TODO / add details.
2. Create `internal/ubuntu/ubuntu.go`:
   ```go
   package ubuntu

   import "github.com/spf13/cobra"

   func Command() *cobra.Command {
       cmd := &cobra.Command{
           Use:   "ubuntu",
           Short: "Switch Ubuntu APT mirror source",
       }
       cmd.AddCommand(
           // config, list, set, unset subcommands
       )
       return cmd
   }
   ```
3. Maintain a `Mirrors` slice in the file, implement `config` / `set` / `list` / `unset` / `test` five actions (naming consistent with pypi), and route via cobra subcommands.
4. Register in `main.go`'s `init()`:
   ```go
   rootCmd.AddCommand(ubuntu.Command())
   ```
   And add `import "github.com/bingym/mirrorctl/internal/ubuntu"` at the top of the file.
5. Run `make` locally and manually test commands like `mirrorctl ubuntu config`.
6. Write `internal/ubuntu/ubuntu_test.go`.
7. Update the "Supported Mirrors" table in `README.md`.

## How to Add a New Mirror (under an existing type)

For example, adding a `douban` mirror to pypi:

1. Open `internal/pypi/pypi.go`, find the `Mirrors` slice.
2. Add an entry before the closing:
   ```go
   {
       Name:        "douban",
       URL:         "https://pypi.douban.com/simple/",
       TrustedHost: "pypi.douban.com",
       Desc:        "Douban",
   },
   ```
3. Run `make` to rebuild, then `mirrorctl pypi list` to verify.
4. Update the pypi mirror table in `README.md`.

**Do not** modify `cmdSet` / `cmdConfig` and other functions; they should be table-driven.

## Known Pitfalls / Notes

- **Do not directly modify system files like `/etc/apt/sources.list`**: When implementing the ubuntu module, always backup the original file to `*.bak` first, and only allow execution under `root` or sudo.
- `~` expansion: Do not hand-roll it; use `util.ExpandHome`, which falls back to `os/user.Current()` when `$HOME` is not set.
- When writing pip config files, if both `~/.config/pip/pip.conf` and `~/.pip/pip.conf` exist, **prefer the XDG path**; only fall back to the legacy path when the XDG path does not exist but the legacy one does.
- `TrustedHost` field: Not all mirrors need it (e.g. only meaningful for https with self-signed certificates); leave it empty in the table to skip writing the `[install]` section.

## Collaboration Conventions

- After modifying code, **make sure to update the TODO check status in `README.md`**.
- When adding new files, add an entry in the "Directory Structure" section of this file.
- Do not delete seemingly verbose explanations in `AGENTS.md` for the sake of "brevity"; these provide context for subsequent agents.
- Commit messages should follow the `<type>: <description>` format, e.g. `feat: implement pypi set subcommand`.

## Migration Notes from C to Go

This project was originally written in C and later fully migrated to Go. Key changes:

| C Version                        | Go Version                         |
|----------------------------------|------------------------------------|
| `src/main.c`                     | `main.go`                          |
| `src/util.h` (header-only)       | `internal/util/util.go`            |
| `src/pypi.{c,h}`                 | `internal/pypi/pypi.go`            |
| `Makefile` (gcc + ld)            | `Makefile` (go build)              |
| Manual malloc/free               | Go GC automatic management         |
| `sbuf_t` dynamic string          | `strings.Builder`                  |
| Custom test framework (planned)  | Standard `testing` package         |
| `strndup` / `memchr` manual parsing | `bufio.Scanner` + `strings` package |

The Go version maintains exactly the same behavior and user experience as the C version, but with safer and more maintainable code.
