package main

import (
	"bytes"
	"testing"
)

func TestEnrollmentCLIRejectsIncompleteOrUnsafeSetup(t *testing.T) {
	for _, args := range [][]string{nil, {"-url", "https://example.invalid"}, {"-url", "https://user:password@example.invalid", "-policy", "/unused", "-state-dir", "/unused"}, {"-unexpected"}} {
		var out, errOut bytes.Buffer
		if e := run(args, &out, &errOut); e == nil {
			t.Fatal("unsafe setup accepted")
		}
		if out.Len() != 0 {
			t.Fatal("output before validated setup")
		}
	}
}
