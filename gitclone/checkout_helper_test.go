package gitclone

import "testing"

func Test_branchRefToTrackingBranch(t *testing.T) {
	tests := []struct {
		name      string
		branchRef string
		want      string
	}{
		{
			"Branch refspec",
			"refs/heads/master",
			"master",
		},
		{
			"Branch with refs in name",
			"refs/heads/refs/master",
			"refs/master",
		},
		{
			"Head/merge branch refspec",
			"refs/pull/1/head",
			"pull/1/head",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := branchRefToTrackingBranch(tt.branchRef); got != tt.want {
				t.Errorf("branchRefToTrackingBranch() = %v, want %v", got, tt.want)
			}
		})
	}
}
