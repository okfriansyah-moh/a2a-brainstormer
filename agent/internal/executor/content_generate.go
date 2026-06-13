package executor

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv"

	"a2a-brainstorm/agent/internal/llm"
)

const maxJSONContinuations = 2

type generatedContent struct {
	text         string
	finishReason string
}

// generateStateContent calls the LLM (streaming first, blocking fallback) and
// auto-continues when the model hits the output token limit or JSON is incomplete.
func (e *BrainstormExecutor) generateStateContent(
	ctx context.Context,
	execCtx *a2asrv.ExecutorContext,
	activeLLM llm.LLMProvider,
	llmReq llm.LLMRequest,
	yield func(a2a.Event, error) bool,
	role string,
) (string, error) {
	first, err := e.generateOnce(ctx, execCtx, activeLLM, llmReq, yield, role)
	if err != nil {
		return "", err
	}

	content := first.text
	finishReason := first.finishReason

	for attempt := 0; attempt < maxJSONContinuations; attempt++ {
		needsContinue := finishReason == "length" || finishReason == "max_tokens"
		if !needsContinue {
			if _, parseErr := extractJSON(content); parseErr == nil {
				break
			}
			needsContinue = true
		}
		if !needsContinue {
			break
		}

		tail := truncate(content, 400)
		contReq := llmReq
		contReq.UserMessage = continueTruncatedJSONPrompt + "\n\nPrior JSON ended with:\n" + tail

		next, err := e.generateOnce(ctx, execCtx, activeLLM, contReq, yield, role)
		if err != nil {
			return content, err
		}
		content += next.text
		finishReason = next.finishReason

		if _, parseErr := extractJSON(content); parseErr == nil {
			break
		}
	}

	if _, err := extractJSON(content); err != nil {
		return content, err
	}
	return content, nil
}

func (e *BrainstormExecutor) generateOnce(
	ctx context.Context,
	execCtx *a2asrv.ExecutorContext,
	activeLLM llm.LLMProvider,
	llmReq llm.LLMRequest,
	yield func(a2a.Event, error) bool,
	role string,
) (generatedContent, error) {
	if sp, streamOK := activeLLM.(llm.StreamingLLMProvider); streamOK {
		out, used, err := e.generateStream(ctx, execCtx, sp, llmReq, yield, role)
		if err != nil {
			return generatedContent{}, err
		}
		if used {
			return out, nil
		}
	}

	resp, err := activeLLM.Generate(ctx, llmReq)
	if err != nil {
		return generatedContent{}, err
	}
	return generatedContent{text: resp.Content, finishReason: resp.FinishReason}, nil
}

func (e *BrainstormExecutor) generateStream(
	ctx context.Context,
	execCtx *a2asrv.ExecutorContext,
	sp llm.StreamingLLMProvider,
	llmReq llm.LLMRequest,
	yield func(a2a.Event, error) bool,
	role string,
) (generatedContent, bool, error) {
	chunks, streamErr := sp.GenerateStream(ctx, llmReq)
	if streamErr != nil {
		return generatedContent{}, false, nil
	}

	var sb strings.Builder
	finishReason := ""
	allOK := true
	for chunk := range chunks {
		if chunk.Err != nil {
			e.logError(ctx, "LLM stream error, falling back to blocking Generate",
				chunk.Err, slog.String("role", role))
			allOK = false
			break
		}
		if chunk.Text != "" {
			sb.WriteString(chunk.Text)
			tokenMsg := a2a.NewMessage(a2a.MessageRoleAgent, a2a.NewTextPart(chunk.Text))
			if !yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateWorking, tokenMsg), nil) {
				return generatedContent{}, true, fmt.Errorf("stream cancelled")
			}
		}
		if chunk.FinishReason != "" {
			finishReason = chunk.FinishReason
		}
	}
	if !allOK {
		return generatedContent{}, false, nil
	}
	return generatedContent{text: sb.String(), finishReason: finishReason}, true, nil
}
