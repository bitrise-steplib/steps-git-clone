#!/bin/env bash
set -ex

#
# Functions
clone_repo() {
    local repo=$1
    local tag=$2
    local __resultvar=$3

    tmpdir=$(mktemp -d 2>/dev/null || mktemp -d -t 'mytmpdir')

    cd $tmpdir
    git init
    git remote add "origin" "$repo"
    git "fetch" "--jobs=10" "--depth=1" "--no-tags" "--no-recurse-submodules" "origin" "refs/tags/$tag:refs/tags/$tag"
    git "checkout" "$tag"
    cd -

    eval $__resultvar="'$tmpdir'"
}

run_becnhmark() {
    local version=$1
    local count=$2
    local __resultvar=$3

    local step_dir=""
    if [[ "$version" == "local" ]]; then
        step_dir="./.."
    else
        clone_repo "https://github.com/bitrise-steplib/steps-git-clone" "$version" step_dir
    fi

    bench_output="$version-benchmark"
    rm -rf "$bench_output"

    export CURRENT_STEP_DIR="$step_dir"

    [ ! -z "$BENCH_COMMIT" ] &&                                     go test -run=XXX -bench="BenchmarkCommitCheckout$" -count="$count" | tee -a "$bench_output"
    [ ! -z "$BENCH_COMMIT" ] && [ ! -z "$BENCH_COMMIT_BRANCH" ] &&  go test -run=XXX -bench="BenchmarkCommitCheckout_branch_specified$" -count="$count" | tee -a "$bench_output"
    [ ! -z "$BENCH_BRANCH" ] &&                                     go test -run=XXX -bench="BenchmarkBranchCheckout$" -count="$count" | tee -a "$bench_output"
    [ ! -z "$BENCH_TAG" ] &&                                        go test -run=XXX -bench="BenchmarkTagCheckout$" -count="$count" | tee -a "$bench_output"
    [ ! -z "$BENCH_TAG" ] && [ ! -z "$BENCH_TAG_BRANCH" ] &&        go test -run=XXX -bench="BenchmarkTagCheckout_branch_specified$" -count="$count" | tee -a "$bench_output"

    eval $__resultvar="'$bench_output'"
}

#
# Benchmark configuration
export BENCH_REPOSITORY_URL="https://github.com/apple/swift.git"
# Commit checkout
export BENCH_COMMIT="ff7e6ad744e48e0465b6bef69a1e385b2b6303c6"
export BENCH_COMMIT_BRANCH="next"
# Tag checkout
export BENCH_TAG="swift-5.4-DEVELOPMENT-SNAPSHOT-2021-03-15-a"
export BENCH_TAG_BRANCH="release/5.4"
# Branch checkout
export BENCH_BRANCH="next"

current_version="local"
previous_version=""
bench_count="3"

#
# Run benchmark against local version
current_bench_output=""
run_becnhmark "$current_version" "$bench_count" current_bench_output

#
# Run benchmark against previous version
if [[ ! -z "$previous_version" ]] ; then
    previous_bench_output=""
    run_becnhmark "$previous_version" "$bench_count" previous_bench_output
fi

#
# Compare benchmarks
if [[ ! -z "$previous_version" ]] ; then
    echo
    echo "$previous_version benchmark:"
    benchstat "$previous_bench_output"
fi

echo
echo "$current_version benchmark:"
benchstat "$current_bench_output"

if [[ ! -z "$previous_version" ]] ; then
    echo
    echo "$previous_version <-> $current_version comparison:"
    benchstat "$previous_bench_output" "$current_bench_output"
fi
