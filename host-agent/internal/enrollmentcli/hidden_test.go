package enrollmentcli

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
)

func TestHiddenInputRejectsNonTerminal(t *testing.T) {
	f, e := os.CreateTemp(t.TempDir(), "input")
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	var output bytes.Buffer
	if _, e := readHiddenBrowserToken(context.Background(), f, &output); e == nil {
		t.Fatal("non-terminal credential input accepted")
	}
}
func TestHiddenInputBoundsAndEditing(t *testing.T) {
	got, e := readHiddenLine(context.Background(), strings.NewReader("YWJjX\b=\r"))
	if e != nil || got != "YWJj=" {
		t.Fatal("hidden editing failed")
	}
	for _, bad := range []string{strings.Repeat("A", 65537) + "\n", "has space\n", "\x1b[200~value\n"} {
		if _, e := readHiddenLine(context.Background(), strings.NewReader(bad)); e == nil {
			t.Fatal("unsafe hidden input accepted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := readHiddenLine(ctx, strings.NewReader("value\n")); e == nil {
		t.Fatal("cancel ignored")
	}
}
