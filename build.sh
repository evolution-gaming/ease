#!/usr/bin/env bash
#
# Helper build script.
#
# Function defined as do_something are intended to be used as runnable actions when this
# script is executed.
#
# Example:
#
#   ./build.sh test
#
# Will dispatch to do_test function as defined in this script.

set -euo pipefail

APP_NAME="ease"
VERSION=$(git describe --tags --abbrev=8 --dirty --always)
OUT_DIR="out"
RELEASE_DIR="${OUT_DIR}/release"
FFPROBE_EXE=$(command -v ffprobe)
FFMPEG_EXE=$(command -v ffmpeg)
GO="${GO:-$(command -v go)}"

# The firs command line argument is the name of action to run. Default action is "help".
ACTION="${1:-help}"

# Helper functions
err() {
    printf "ERROR: %s\n" "$*" 1>&2
}

log() {
    printf "%s\n" "$*"
}

# Define ACTION functions in form do_<action>. From commanline point of view the "do_"
# part is stripped.

# Help on usage.
do_help() {
    log "Supported build actions:"   
    awk '
    /^do_[a-zA-Z0-9_]+\(\)/ {
        a=$0;
        gsub("[(){]", "", a);
        gsub("^do_", "", a);
        printf "%-15s %s\n", a, prev;
    }
    {
        prev=$0;
    }
    ' "$0"
}

# Run tests.
do_test() {
    if [ -z "$FFPROBE_EXE" ] ; then
        err "ffprobe not found in PATH, ffprobe is required for ease tool and tests"
        exit 1
    fi
    if [ -z "$FFMPEG_EXE" ] ; then
        err "ffmpeg not found in PATH, ffmpeg is required for ease tool and tests"
        exit 1
    fi

    "$GO" test -cover ./...
}

# Create test coverage report HTML.
do_coverage() {
    mkdir -p out
    "$GO" test -coverprofile="${OUT_DIR}"/coverage.out ./... \
    && "$GO" tool cover -html="${OUT_DIR}"/coverage.out
}

# Run linters and static code analysis checks.
do_lint() {
    golangci-lint run
}

# Create build.
do_build() {
    CGO_ENABLED=0 "$GO" build -o "${OUT_DIR}/${APP_NAME}" -trimpath \
        -ldflags="-X main.version=$VERSION -buildid=" -v

    build_artifact_info "${OUT_DIR}/${APP_NAME}"
}

# Release build includes additional flags.
rel_build() {
    CGO_ENABLED=0 "$GO" build -o "${OUT_DIR}/${APP_NAME}" -trimpath \
        -ldflags="-s -w -X main.version=$VERSION -buildid=" -v

    build_artifact_info "${OUT_DIR}/${APP_NAME}"
}

# Print some relevant info on build artifact.
#
# arg $1: path to artifact/binary
build_artifact_info() {
    go version -m "$1"
    stat "$1"

    # On GitHub CI also set step output
    if [ "${GITHUB_ACTIONS:-false}" == "true" ] ; then
        echo "::set-output name=artifact_path::$1"
    fi
}

# Generate a simple changelog from git commits
git_changelog() {
    local to_tag from_tag
    declare -a tags

    tags=($(git for-each-ref refs/tags/* --sort=-taggerdate --count=2 --format="%(refname:short)"))
    to_tag=${tags[0]}
    from_tag=${tags[1]}
    echo "## Changelog ($to_tag)"
    git log  --pretty=format:'* %h %s' "${from_tag}".."${to_tag}"
    echo
}

# Prepare release artifacts.
do_release() {
    log "Removing old artifacts"
    do_clean
    
    mkdir -p "$RELEASE_DIR"

    # For Linux we create just amd64 arch build
    GOOS=linux GOARCH=amd64 rel_build
    tar --gzip -cvf "${RELEASE_DIR}/${APP_NAME}-linux-amd64.tar.gz" -C "$OUT_DIR" "$APP_NAME"

    # For macOS create both amd64 and arm64 builds
    GOOS=darwin GOARCH=amd64 rel_build
    zip -j "${RELEASE_DIR}/${APP_NAME}-darwin-amd64.zip" "${OUT_DIR}/${APP_NAME}"

    GOOS=darwin GOARCH=arm64 rel_build
    zip -j "${RELEASE_DIR}/${APP_NAME}-darwin-arm64.zip" "${OUT_DIR}/${APP_NAME}"

    # Create checksums file
    pushd "$RELEASE_DIR"
    md5sum "$APP_NAME"* >md5sums.txt
    git_changelog > changelog.txt
    popd
}

# Clean build artifacts.
do_clean() {
    "$GO" clean
    rm -v -rf "${OUT_DIR:?}"/*

}

# Dispatch to defined ACTION function.
if grep -q "do_${ACTION}()" "$0" ; then
    log "== Running $ACTION =="
    "do_${ACTION}"
else
    err "Action $ACTION unknown"
    do_help
fi
