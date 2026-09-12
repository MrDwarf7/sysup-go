# justfile for sysup-go
# Module path is sysup-go; binaries: sysup-dbg (debug), sysup (release), sysup-expr (experimental).
# Debug is the strict default (race, no-opt, full vet). Release is stripped, no race.
# Experimental: extra pointer checks at compile, GC off + clobber/efence at run.

set shell := ["bash", "-cu"]

bindir := "bin"
bin_dbg := bindir / "sysup-dbg"
bin_rel := bindir / "sysup"
bin_expr := bindir / "sysup-expr"
pkg := "./..."
testpkg := "./tests"
coverprofile := bindir / "coverage.out"
ldflags_release := "-s -w -buildid="
# -N -l: no optimize / no inline (debuggable). checkptr is on with -race.
gcflags_debug := "all=-N -l"
# checkptr=2: extra invalid-pointer checks (may false-positive). No race: races vs asan/efence.
gcflags_expr := "all=-N -l -d=checkptr=2"
race_env := "GORACE=halt_on_error=1"
# GOGC=off really disables GC. clobberfree/efence/invalidptr are the closest
# thing Go has to "use-after-free / dangling" traps. cgocheck2 is a build-time GOEXPERIMENT.
expr_env := "GOGC=off GODEBUG=clobberfree=1,invalidptr=1,gccheckmark=1,efence=1,gctrace=1"

# list recipes
default:
    @just --list

_ensure-bindir:
    mkdir -p {{ bindir }}

# build debug binary (b) -> bin/sysup-dbg  (race + no-opt)
alias b := build
build: _ensure-bindir
    {{ race_env }} go build -race -gcflags "{{ gcflags_debug }}" -o {{ bin_dbg }} .

# build optimized release binary (br) -> bin/sysup
# CGO off, stripped, trimpath, no VCS stamp, readonly modules, no race
alias br := build-release
build-release: _ensure-bindir
    CGO_ENABLED=0 go build -mod=readonly -ldflags "{{ ldflags_release }}" -trimpath -buildvcs=false -o {{ bin_rel }} .

# build experimental binary (be) -> bin/sysup-expr
# compile-time checkptr=2; run with `just re` for GOGC=off + clobber/efence.
# Do not add GOEXPERIMENT=cgocheck2 here: it panics at startup with efence
# ("unpinned Go pointer stored into non-Go memory" in runtime.parsegodebug).
alias be := build-expr
build-expr: _ensure-bindir
    go build -gcflags "{{ gcflags_expr }}" -o {{ bin_expr }} .

# build debug + release and print size delta (ba)
alias ba := build-all
build-all: build build-release
    @dbg=$(stat -c%s {{ bin_dbg }}); rel=$(stat -c%s {{ bin_rel }}); \
    printf 'debug    %10d  %s\n' "$dbg" "{{ bin_dbg }}"; \
    printf 'release  %10d  %s\n' "$rel" "{{ bin_rel }}"; \
    printf 'saved    %10d\n' "$((dbg - rel))"; \
    if [ -f {{ bin_expr }} ]; then \
      printf 'expr     %10d  %s\n' "$(stat -c%s {{ bin_expr }})" "{{ bin_expr }}"; \
    fi

# static analysis (c): golangci-lint default:all
alias c := check
check:
    go tool golangci-lint run {{ pkg }}

# remove build artifacts (cl)
alias cl := clean
clean:
    rm -rf {{ bindir }}
    go clean

# format all Go sources (f)
alias f := format
format: taplo-format
    go fmt {{ pkg }}

# format TOML files via taplo
taplo-format:
    taplo format

# fail if any file is not gofmt (CI)
format-check:
    files="$(gofmt -l .)"; if [ -n "$files" ]; then printf '%s\n' "$files"; exit 1; fi

# pulls and checks known vulnerabilities from official source(s)
vuln:
    go run golang.org/x/vuln/cmd/govulncheck@latest ./...

# run unit tests (t). extra args pass through: `just t -v`
# race, shuffle, full vet, halt on first race. always debug-class flags.
alias t := test
test *args:
    {{ race_env }} go test -race -count=1 -shuffle=on -timeout 2m -vet=all {{ testpkg }} {{ args }}

# coverage (atomic required with -race); profile stays under bin/
cover: _ensure-bindir
    {{ race_env }} go test -race -count=1 -shuffle=on -timeout 2m -vet=all -covermode=atomic -coverpkg={{ pkg }} -coverprofile={{ coverprofile }} {{ testpkg }}
    go tool cover -func={{ coverprofile }}

# debug-build then run: `just r list`
alias r := run
run *args: build
    {{ race_env }} ./{{ bin_dbg }} {{ args }}

# release-build then run: `just rr list`
alias rr := run-release
run-release *args: build-release
    ./{{ bin_rel }} {{ args }}

# experimental-build then run: `just re list`
# GC off, freed memory clobbered, unique pages (efence), extra pointer checks.
alias re := run-expr
run-expr *args: build-expr
    {{ expr_env }} ./{{ bin_expr }} {{ args }}

# full pipeline (a): format -> check -> test -> debug+release
alias a := all
all: format check test build-all

# clean then full pipeline - no alias by design
reset: clean all
