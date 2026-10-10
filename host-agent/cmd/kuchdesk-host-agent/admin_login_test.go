package main

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestAdminLoginUsesConfiguredAbsoluteRoot(t *testing.T) {
	t.Setenv("KUCHDESK_PROJECT_ROOT", "/configured/project with spaces")
	for _, tc := range []struct {
		args []string
		root string
	}{
		{nil, "/configured/project with spaces"},
		{[]string{"--project-root", "/explicit/project"}, "/explicit/project"},
	} {
		var out bytes.Buffer
		sentinel := errors.New("authentication_denied")
		err := runAdminLogin(tc.args, &out, &out, func(args []string, stdout, stderr io.Writer) error {
			if !reflect.DeepEqual(args, []string{"-login-easy", "-project-root", tc.root}) {
				t.Fatal("wrong shared login arguments", args)
			}
			return sentinel
		})
		if !errors.Is(err, sentinel) {
			t.Fatal("authentication error lost", err)
		}
	}
}
func TestAdminLoginHelpAndInvalidConfigurationNeverLaunch(t *testing.T) {
	t.Setenv("KUCHDESK_PROJECT_ROOT", "")
	for _, args := range [][]string{{"--help"}, nil, {"--project-root", "relative"}, {"unexpected"}, {"--url", "https://other.test"}} {
		var out bytes.Buffer
		err := runAdminLogin(args, &out, &out, func([]string, io.Writer, io.Writer) error { t.Fatal("unexpected login launch"); return nil })
		if len(args) > 0 && args[0] == "--help" {
			if err != nil || !strings.Contains(out.String(), "KUCHDESK_PROJECT_ROOT") {
				t.Fatal("help unavailable", err)
			}
		} else if err == nil {
			t.Fatal("invalid configuration accepted")
		}
	}
}
