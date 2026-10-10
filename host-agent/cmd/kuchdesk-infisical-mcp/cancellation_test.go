package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/async"
	"github.com/amirtaherkhani/kuchdesk/host-agent/internal/infisical"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type blockingNameLister struct{ entered chan string }

func (b blockingNameLister) Configured() bool { return true }
func (b blockingNameLister) ListSecretNames(ctx context.Context, project, _, _ string) ([]infisical.SecretName, error) {
	b.entered <- project
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestMCPCancelsQueuedAndRunningJobsThroughTransport(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	runner := async.NewRunner(ctx, 1, 4)
	defer runner.Close()
	lister := blockingNameLister{entered: make(chan string, 3)}
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	server, err := newServerWithRunner(infisical.NewService(lister), runner).Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	client, err := mcp.NewClient(&mcp.Implementation{Name: "cancellation-test"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	call := func(name string, args map[string]any, out any) {
		t.Helper()
		result, err := client.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		if err != nil {
			t.Fatal(err)
		}
		if result.IsError {
			t.Fatalf("%s unexpectedly rejected", name)
		}
		encoded, err := json.Marshal(result.StructuredContent)
		if err != nil {
			t.Fatal(err)
		}
		if err = json.Unmarshal(encoded, out); err != nil {
			t.Fatal(err)
		}
	}
	start := func(project string) string {
		t.Helper()
		var out jobStarted
		call("infisical_start_list_secret_names", map[string]any{"projectId": project, "environment": "dev", "secretPath": "/app"}, &out)
		if out.ID == "" {
			t.Fatal("missing job ID")
		}
		return out.ID
	}
	awaitEntry := func(want string) {
		t.Helper()
		select {
		case got := <-lister.entered:
			if got != want {
				t.Fatal("queued canceled operation executed")
			}
		case <-ctx.Done():
			t.Fatal("operation did not enter")
		}
	}
	awaitCanceled := func(id string) {
		t.Helper()
		for {
			var out jobOutput
			call("infisical_job_status", map[string]any{"id": id}, &out)
			if out.State == async.Canceled {
				if len(out.Secrets) != 0 {
					t.Fatal("canceled job retained result")
				}
				return
			}
			select {
			case <-ctx.Done():
				t.Fatal("cancellation did not complete")
			case <-time.After(time.Millisecond):
			}
		}
	}
	first := start("running")
	awaitEntry("running")
	queued := start("queued")
	var out jobOutput
	call("infisical_cancel_job", map[string]any{"id": queued}, &out)
	call("infisical_cancel_job", map[string]any{"id": first}, &out)
	awaitCanceled(first)
	awaitCanceled(queued)
	next := start("after-cancel")
	awaitEntry("after-cancel")
	call("infisical_cancel_job", map[string]any{"id": next}, &out)
	awaitCanceled(next)
}
