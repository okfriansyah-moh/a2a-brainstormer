package a2a

import (
	"errors"
	"io"
	"testing"

	sdk "github.com/a2aproject/a2a-go/v2/a2a"
)

func TestTerminalTaskResult_WorkingNotTerminal(t *testing.T) {
	task := &sdk.Task{Status: sdk.TaskStatus{State: sdk.TaskStateWorking}}
	_, _, done := terminalTaskResult(task, nil)
	if done {
		t.Fatal("expected in-progress task to be non-terminal")
	}
}

func TestTerminalTaskResult_CompletedWithArtifact(t *testing.T) {
	want := map[string]any{"metrics": map[string]any{"confidence": 0.5}}
	task := &sdk.Task{
		Status: sdk.TaskStatus{State: sdk.TaskStateCompleted},
		Artifacts: []*sdk.Artifact{
			{Parts: sdk.ContentParts{sdk.NewDataPart(want)}},
		},
	}
	result, err, done := terminalTaskResult(task, nil)
	if !done || err != nil {
		t.Fatalf("done=%v err=%v", done, err)
	}
	got, extractErr := ExtractStateFromResult(result)
	if extractErr != nil {
		t.Fatalf("ExtractStateFromResult() error = %v", extractErr)
	}
	if got == nil {
		t.Fatal("expected artifact data")
	}
}

func TestTerminalTaskResult_CompletedUsesStreamArtifact(t *testing.T) {
	streamed := map[string]any{"idea": map[string]any{"text": "delta"}}
	task := &sdk.Task{Status: sdk.TaskStatus{State: sdk.TaskStateCompleted}}
	result, err, done := terminalTaskResult(task, streamed)
	if !done || err != nil {
		t.Fatalf("done=%v err=%v", done, err)
	}
	got, extractErr := ExtractStateFromResult(result)
	if extractErr != nil {
		t.Fatalf("ExtractStateFromResult() error = %v", extractErr)
	}
	if got == nil {
		t.Fatal("expected streamed artifact data")
	}
}

func TestTerminalTaskResult_Failed(t *testing.T) {
	task := &sdk.Task{
		Status: sdk.TaskStatus{
			State:   sdk.TaskStateFailed,
			Message: sdk.NewMessage(sdk.MessageRoleAgent, sdk.NewTextPart("LLM returned non-JSON")),
		},
	}
	_, err, done := terminalTaskResult(task, nil)
	if !done || err == nil {
		t.Fatalf("done=%v err=%v", done, err)
	}
	if !errors.Is(err, err) {
		if err.Error() == "" {
			t.Fatal("expected failure error message")
		}
	}
}

func TestTerminalTaskResult_CompletedWithoutArtifact(t *testing.T) {
	task := &sdk.Task{Status: sdk.TaskStatus{State: sdk.TaskStateCompleted}}
	_, err, done := terminalTaskResult(task, nil)
	if !done || err == nil {
		t.Fatalf("done=%v err=%v", done, err)
	}
}

func TestIsRecoverableStreamClose(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{io.EOF, true},
		{io.ErrUnexpectedEOF, true},
		{errors.New("SSE stream error: unexpected EOF"), true},
		{errors.New("context canceled"), false},
	}
	for _, tc := range cases {
		if got := isRecoverableStreamClose(tc.err); got != tc.want {
			t.Errorf("isRecoverableStreamClose(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
}
