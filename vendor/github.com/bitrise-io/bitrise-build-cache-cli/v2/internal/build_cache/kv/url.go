package kv

import (
	"fmt"
	"net/url"
)

func ParseURLGRPC(s string) (string, bool, error) {
	parsed, err := url.ParseRequestURI(s)
	if err != nil {
		return "", false, fmt.Errorf("parse url: %w", err)
	}

	isSecure := parsed.Scheme == "grpcs"

	if parsed.Scheme != "grpc" && parsed.Scheme != "grpcs" {
		return "", false, fmt.Errorf("scheme must be grpc or grpcs")
	}

	host := parsed.Host
	if parsed.Port() == "" {
		if isSecure {
			host += ":443"
		} else {
			host += ":80"
		}
	}

	return host, !isSecure, nil
}
