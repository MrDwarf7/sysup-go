# justfile for sysup-go
# Slim, idiomatic Go task runner. `just` lists recipes; aliases in parens.
# Prefer `just <recipe>` over bare `go` invocations so flags stay in one place.

set shell := ["bash", "-cu"]

bin := "bin/sysup-go"
pkg := "./..."
# Black-box tests live in tests/ (package tests), not next to source.
testpkg := "./tests"
# -s -w strip symbol/DWARF tables; empty buildid is reproducible.
ldflags_release := "-s -w -buildid="

# list recipes
default:
    @just --list

# build debug binary (b)
alias b := build
build:
    go build -o {{bin}} .

# build optimized release binary (br)
# CGO off, stripped, trimpath, no VCS stamp, readonly modules
alias br := build-release
build-release:
    CGO_ENABLED=0 go build -mod=readonly -ldflags "{{ldflags_release}}" -trimpath -buildvcs=false -o {{bin}} .

# static analysis (c) - keep slim: go vet only
alias c := check
check:
    go vet {{pkg}}

# remove build artifacts (cl)
alias cl := clean
clean:
    rm -rf bin coverage.out
    go clean

# format all Go sources (f)
alias f := format
format:
    go fmt {{pkg}}

# run unit tests (t). extra args pass through: `just t -v`
alias t := test
test *args:
    go test {{testpkg}} -count=1 {{args}}

# coverage across production packages (tests are a separate package)
cover:
    go test {{testpkg}} -count=1 -coverpkg={{pkg}} -coverprofile=coverage.out
    go tool cover -func=coverage.out

# debug-build then run the binary: `just run list`
run *args: build
    ./{{bin}} {{args}}

# full pipeline (a): format -> check -> test -> build
alias a := all
all: format check test build

# clean then full pipeline - no alias by design
reset: clean all
