package upstream

import (
	"errors"
	"testing"
)

func TestIsRetryable_http429And5xx(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{errors.New("network"), false},
		{&HTTPError{Status: 400, Body: "bad request"}, false},
		{&HTTPError{Status: 404, Body: "not found"}, false},
		{&HTTPError{Status: 429, Body: "rate limit"}, true},
		{&HTTPError{Status: 500, Body: "internal"}, true},
		{&HTTPError{Status: 502, Body: "bad gateway"}, true},
		{&HTTPError{Status: 503, Body: "unavailable"}, true},
	}
	for _, tc := range cases {
		got := IsRetryable(tc.err)
		if got != tc.want {
			t.Errorf("IsRetryable(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
}
