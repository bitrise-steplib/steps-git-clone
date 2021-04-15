package bench

import (
	"os"
	"testing"
)

var localStepPath = os.Getenv("CURRENT_STEP_DIR")
var repositoryURL = os.Getenv("BENCH_REPOSITORY_URL")

func BenchmarkCommitCheckout(b *testing.B) {
	for n := 0; n < b.N; n++ {
		commit := os.Getenv("BENCH_COMMIT")
		tag := ""
		branch := ""

		if err := bitriseRun(localStepPath, repositoryURL, commit, tag, branch); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCommitCheckout_branch_specified(b *testing.B) {
	for n := 0; n < b.N; n++ {
		commit := os.Getenv("BENCH_COMMIT")
		tag := ""
		branch := os.Getenv("BENCH_COMMIT_BRANCH")

		if err := bitriseRun(localStepPath, repositoryURL, commit, tag, branch); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBranchCheckout(b *testing.B) {
	for n := 0; n < b.N; n++ {
		commit := ""
		tag := ""
		branch := os.Getenv("BENCH_BRANCH")

		if err := bitriseRun(localStepPath, repositoryURL, commit, tag, branch); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTagCheckout(b *testing.B) {
	for n := 0; n < b.N; n++ {
		commit := ""
		tag := os.Getenv("BENCH_TAG")
		branch := ""

		if err := bitriseRun(localStepPath, repositoryURL, commit, tag, branch); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTagCheckout_branch_specified(b *testing.B) {
	for n := 0; n < b.N; n++ {
		commit := ""
		tag := os.Getenv("BENCH_TAG")
		branch := os.Getenv("BENCH_TAG_BRANCH")

		if err := bitriseRun(localStepPath, repositoryURL, commit, tag, branch); err != nil {
			b.Fatal(err)
		}
	}
}
