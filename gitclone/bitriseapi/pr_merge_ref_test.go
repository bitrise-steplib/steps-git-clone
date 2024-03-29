package bitriseapi

import (
	"fmt"
	"testing"
	"time"

	"github.com/bitrise-io/go-utils/v2/log"
)

func Test_doPoll(t *testing.T) {
	tests := []struct {
		name    string
		fetcher mergeRefFetcher
		want    bool
		wantErr bool
	}{
		{
			name: "Up-to-date for the first check",
			fetcher: func(attempt uint) (mergeRefResponse, error) {
				return mergeRefResponse{Status: "up-to-date"}, nil
			},
			want: true,
		},
		{
			name: "Pending for first check, up-to-date for second",
			fetcher: func(attempt uint) (mergeRefResponse, error) {
				if attempt == 0 {
					return mergeRefResponse{Status: "pending"}, nil
				}
				return mergeRefResponse{Status: "up-to-date"}, nil
			},
			want: true,
		},
		{
			name: "Unrecoverable error for first check",
			fetcher: func(attempt uint) (mergeRefResponse, error) {
				return mergeRefResponse{Error: "A fatal error", ShouldRetry: false}, nil
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "Network error for first check, success for second",
			fetcher: func(attempt uint) (mergeRefResponse, error) {
				if attempt == 0 {
					return mergeRefResponse{}, fmt.Errorf("mocked network error")
				}
				return mergeRefResponse{Status: "up-to-date"}, nil
			},
			want: true,
		},
		{
			name: "Exceeding retries, result is still pending",
			fetcher: func(attempt uint) (mergeRefResponse, error) {
				return mergeRefResponse{Status: "pending"}, nil
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "Unknown status for first check, success for second",
			fetcher: func(attempt uint) (mergeRefResponse, error) {
				if attempt == 0 {
					return mergeRefResponse{Status: "unknown"}, nil
				}
				return mergeRefResponse{Status: "up-to-date"}, nil
			},
			want: true,
		},
		{
			name: "Error message in response",
			fetcher: func(attempt uint) (mergeRefResponse, error) {
				return mergeRefResponse{Error: "Test error message"}, nil
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "Not mergeable PR state",
			fetcher: func(attempt uint) (mergeRefResponse, error) {
				return mergeRefResponse{Status: "not-mergeable"}, nil
			},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retryWaitTime := time.Duration(0)
			maxAttemptCount := uint(5)
			got, _, err := doPoll(tt.fetcher, maxAttemptCount, retryWaitTime, log.NewLogger())
			if (err != nil) != tt.wantErr {
				t.Errorf("doPoll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("doPoll() got = %v, want %v", got, tt.want)
			}
		})
	}
}
