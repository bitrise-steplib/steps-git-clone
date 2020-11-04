package gitclone

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/bitrise-io/bitrise-init/step"
)

func Test_newDetailedErrorRecommendation(t *testing.T) {
	type args struct {
		detailedError DetailedError
	}
	tests := []struct {
		name string
		args args
		want step.Recommendation
	}{
		{
			name: "newDetailedErrorRecommendation with nil",
			args: args{
				detailedError: DetailedError{
					Title:       "TestTitle",
					Description: "TestDesciption",
				},
			},
			want: step.Recommendation{
				detailedErrorRecKey: DetailedError{
					Title:       "TestTitle",
					Description: "TestDesciption",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newDetailedErrorRecommendation(tt.args.detailedError); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newDetailedErrorRecommendation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getParamAt(t *testing.T) {
	type args struct {
		index  int
		params []string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "getParamsAt(0, nil)",
			args: args{
				index:  0,
				params: nil,
			},
			want: unknownParam,
		},
		{
			name: "getParamsAt(0, [])",
			args: args{
				index:  0,
				params: []string{},
			},
			want: unknownParam,
		},
		{
			name: "getParamsAt(-1, ['1', '2', '3', '4', '5'])",
			args: args{
				index:  -1,
				params: []string{"1", "2", "3", "4", "5"},
			},
			want: unknownParam,
		},
		{
			name: "getParamsAt(5, ['1', '2', '3', '4', '5'])",
			args: args{
				index:  5,
				params: []string{"1", "2", "3", "4", "5"},
			},
			want: unknownParam,
		},
		{
			name: "getParamsAt(0, ['1', '2', '3', '4', '5'])",
			args: args{
				index:  0,
				params: []string{"1", "2", "3", "4", "5"},
			},
			want: "1",
		},
		{
			name: "getParamsAt(4, ['1', '2', '3', '4', '5'])",
			args: args{
				index:  4,
				params: []string{"1", "2", "3", "4", "5"},
			},
			want: "5",
		},
		{
			name: "getParamsAt(2, ['1', '2', '3', '4', '5'])",
			args: args{
				index:  2,
				params: []string{"1", "2", "3", "4", "5"},
			},
			want: "3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getParamAt(tt.args.index, tt.args.params); got != tt.want {
				t.Errorf("getParamAt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPatternErrorMatcher_Run(t *testing.T) {
	type fields struct {
		defaultBuilder   DetailedErrorBuilder
		patternToBuilder PatternToDetailedErrorBuilder
	}
	type args struct {
		msg string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   step.Recommendation
	}{
		{
			name: "Run with defaultBuilder",
			fields: fields{
				defaultBuilder: func(params ...string) DetailedError {
					return DetailedError{
						Title:       "T",
						Description: "D",
					}
				},
				patternToBuilder: map[string]DetailedErrorBuilder{},
			},
			args: args{
				msg: "Test",
			},
			want: step.Recommendation{
				detailedErrorRecKey: DetailedError{
					Title:       "T",
					Description: "D",
				},
			},
		},
		{
			name: "Run with patternBuilder",
			fields: fields{
				defaultBuilder: func(params ...string) DetailedError {
					return DetailedError{
						Title:       "DefaultTitle",
						Description: "DefaultDesc",
					}
				},
				patternToBuilder: map[string]DetailedErrorBuilder{
					"Test": func(params ...string) DetailedError {
						return DetailedError{
							Title:       "PatternTitle",
							Description: "PatternDesc",
						}
					},
				},
			},
			args: args{
				msg: "Test",
			},
			want: step.Recommendation{
				detailedErrorRecKey: DetailedError{
					Title:       "PatternTitle",
					Description: "PatternDesc",
				},
			},
		},
		{
			name: "Run with patternBuilder with param",
			fields: fields{
				defaultBuilder: func(params ...string) DetailedError {
					return DetailedError{
						Title:       "DefaultTitle",
						Description: "DefaultDesc",
					}
				},
				patternToBuilder: map[string]DetailedErrorBuilder{
					"Test (.+)!": func(params ...string) DetailedError {
						p := getParamAt(0, params)
						return DetailedError{
							Title:       "PatternTitle",
							Description: fmt.Sprintf("PatternDesc: '%s'", p),
						}
					},
				},
			},
			args: args{
				msg: "Test WithPatternParam!",
			},
			want: step.Recommendation{
				detailedErrorRecKey: DetailedError{
					Title:       "PatternTitle",
					Description: "PatternDesc: 'WithPatternParam'",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &PatternErrorMatcher{
				defaultBuilder:   tt.fields.defaultBuilder,
				patternToBuilder: tt.fields.patternToBuilder,
			}
			if got := m.Run(tt.args.msg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PatternErrorMatcher.Run() = %v, want %v", got, tt.want)
			}
		})
	}
}
