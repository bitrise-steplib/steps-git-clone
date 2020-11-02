package gitclone

import (
	"errors"
	"reflect"
	"testing"

	"github.com/bitrise-io/bitrise-init/step"
)

func Test_mapRecommendation(t *testing.T) {
	type args struct {
		tag string
		err error
	}
	tests := []struct {
		name string
		args args
		want step.Recommendation
	}{
		{
			name: "checkout_failed generic error mapping",
			args: args{tag: "checkout_failed", err: errors.New("error: pathspec 'master' did not match any file(s) known to git.")},
			want: newDetailedErrorRecommendation(Detail{Title: "We couldn’t checkout your branch.", Description: `Our auto-configurator returned the following error:
error: pathspec 'master' did not match any file(s) known to git.`}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mapRecommendation(tt.args.tag, tt.args.err); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mapRecommendation() = %v, want %v", got, tt.want)
			}
		})
	}
}
