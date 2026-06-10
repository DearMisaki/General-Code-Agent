package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"mewcode/internal/config"
	"mewcode/internal/conversation"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
	"github.com/openai/openai-go/shared"
)

const openaiCompatStreamIdleTimeout = 5 * time.Minute

type openaiCompatClient struct {
	client       openai.Client
	model        string
	systemPrompt string
}

func newOpenAICompatClient(cfg *config.ProviderConfig, systemPrompt string) (*openaiCompatClient, error) {
	apiKey := cfg.ResolveAPIKey()
	if apiKey == "" {
		return nil, &AuthenticationError{
			Message: "OpenAI-compatible API key not found. Set it in .mewcode/config.yaml or via OPENAI_API_KEY env var.",
		}
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(cfg.BaseURL),
	)

	return &openaiCompatClient{
		client:       client,
		model:        cfg.Model,
		systemPrompt: systemPrompt,
	}, nil
}

func (c *openaiCompatClient) Stream(ctx context.Context, conv *conversation.Manager, toolSchemas []map[string]any) (<-chan StreamEvent, <-chan error) {
	events := make(chan StreamEvent, 64)
	errs := make(chan error, 1)

	messages := buildChatCompletionMessages(c.systemPrompt, conv.GetMessages())

	// 向 请求参数追加 可用 tools
	var tools []openai.ChatCompletionToolParam
	for _, s := range toolSchemas {
		name, _ := s["name"].(string)
		desc, _ := s["description"].(string)
		params, _ := s["parameters"].(map[string]any)
		tools = append(tools, openai.ChatCompletionToolParam{
			Function: shared.FunctionDefinitionParam{
				Name:        name,
				Description: param.NewOpt(desc),
				Parameters:  shared.FunctionParameters(params),
				Strict:      param.NewOpt(false),
			},
		})
	}

	// Stream 函数直接返回 channel，使用下面的新协程执行对应的 sse 处理
	go func() {
		defer close(events)
		defer close(errs)

		reqParams := openai.ChatCompletionNewParams{
			Model:    c.model,
			Messages: messages,
			StreamOptions: openai.ChatCompletionStreamOptionsParam{
				IncludeUsage: param.NewOpt(true),
			},
		}
		if len(tools) > 0 {
			reqParams.Tools = tools
		}

		// 调用 openai 接口 创建 sse 流
		stream := c.client.Chat.Completions.NewStreaming(ctx, reqParams)
		defer stream.Close()

		// 由于开启了流式响应，那么工具的信息可能会在多个 sse 包中，所以这里要渐进式的补全信息
		type toolCallAccum struct {
			id       string
			name     string
			argsJSON string
		}
		toolCalls := make(map[int64]*toolCallAccum)

		// 用于 sse 状态推进
		type sseResult struct {
			hasNext bool
		}
		nextCh := make(chan sseResult, 1)

		readNext := func() {
			nextCh <- sseResult{hasNext: stream.Next()}
		}

		// sse 超时定时器
		idle := time.NewTimer(openaiCompatStreamIdleTimeout)
		defer idle.Stop()

		// 循环读 sse，直到 sse 正常关闭 或者 超时
		go readNext()
		for {
			var res sseResult
			select {
			// sse 恢复 [DONE]
			case <-ctx.Done():
				errs <- &NetworkError{Message: fmt.Sprintf("context cancelled: %v", ctx.Err())}
				return
			// 定时器超时
			case <-idle.C:
				errs <- &NetworkError{Message: fmt.Sprintf("stream idle timeout: no SSE events for %s", openaiCompatStreamIdleTimeout)}
				return
			// 收到新数据
			case res = <-nextCh:
			}

			// 重置定时器
			if !idle.Stop() {
				select {
				case <-idle.C:
				default:
				}
			}
			idle.Reset(openaiCompatStreamIdleTimeout)

			if !res.hasNext {
				break
			}

			chunk := stream.Current()

			// 如果 sse json 中存在 usage 字段，代表模型输出结束
			if chunk.JSON.Usage.Valid() && chunk.Usage.PromptTokens != 0 {
				cached := int(chunk.Usage.PromptTokensDetails.CachedTokens)

				// json 中 prompt_tokens 字段包含 cached token 数量，这里减去避免重复计算
				input := int(chunk.Usage.PromptTokens) - cached
				if input < 0 {
					input = 0
				}
				// 结束事件
				events <- StreamEnd{
					StopReason: "end_turn",
					Usage: UsageInfo{
						InputTokens:     input,
						OutputTokens:    int(chunk.Usage.CompletionTokens),
						CacheReadTokens: cached,
					},
				}
				// sse 最后会发送一个 [DONE]，这里继续 readNext，下一次循环交给 select 处理
				go readNext()
				continue
			}

			// 实际数据都在 choice 中
			if len(chunk.Choices) == 0 {
				go readNext()
				continue
			}

			// 前面构造 ChatCompletionNewParams 的时候可以传入一次性生成回答的数量
			// 因为没有传入这个参数，所以每次生成一个回答，只用取 Choices[0]
			choice := chunk.Choices[0]
			delta := choice.Delta

			// delta 中的模型输出 对应 json 中的 content
			if delta.Content != "" {
				events <- TextDelta{Text: delta.Content}
			}

			// delta 中的 工具调用 输出，对应 tool_calls
			for _, tc := range delta.ToolCalls {
				acc, exists := toolCalls[tc.Index]

				// 第一次收到工具调用相关信息
				if !exists {
					acc = &toolCallAccum{}
					toolCalls[tc.Index] = acc
				}

				// 仅仅会在第一个 工具调用 chunk 中会包含唯一 ID，name
				if tc.ID != "" {
					acc.id = tc.ID
				}
				if tc.Function.Name != "" {
					acc.name = tc.Function.Name
					events <- ToolCallStart{ToolName: acc.name, ToolID: acc.id}
				}

				// 参数片段是追加的形式
				if tc.Function.Arguments != "" {
					acc.argsJSON += tc.Function.Arguments
					events <- ToolCallDelta{Text: tc.Function.Arguments}
				}
			}

			// 部分服务工具信息传输结束后，会在 final_reason 字段中指明 tool_calls
			// 部分服务会在 所有工具信息传输完成 后统一指明 stop
			if choice.FinishReason == "tool_calls" || choice.FinishReason == "stop" {
				// 本次工具调用结束，遍历此次调用的全部工具，将工具信息传递给管道
				for _, acc := range toolCalls {
					var args map[string]any
					if acc.argsJSON != "" {
						json.Unmarshal([]byte(acc.argsJSON), &args)
					}
					if args == nil {
						args = map[string]any{}
					}
					events <- ToolCallComplete{
						ToolID:    acc.id,
						ToolName:  acc.name,
						Arguments: args,
					}
				}

				// 这里清空本次所有的工具信息
				toolCalls = make(map[int64]*toolCallAccum)

				// 这里发送 StreamEnd 不代表 sse 真的结束，还需要等待服务端发送 usage 等信息
				if choice.FinishReason == "stop" && !chunk.JSON.Usage.Valid() {
					events <- StreamEnd{StopReason: "end_turn", Usage: UsageInfo{}}
				}
			}

			go readNext()
		}

		if err := stream.Err(); err != nil {
			errs <- classifyOpenAIError(err)
		}
	}()

	return events, errs
}

// buildChatCompletionMessages converts conversation history into the Chat Completions
// message format. The system prompt becomes a system message at the start. Thinking
// blocks are skipped because Chat Completions does not support them natively.
func buildChatCompletionMessages(systemPrompt string, messages []conversation.Message) []openai.ChatCompletionMessageParamUnion {
	var result []openai.ChatCompletionMessageParamUnion

	// System prompt as the first message
	if systemPrompt != "" {
		result = append(result, openai.SystemMessage(systemPrompt))
	}

	for _, m := range messages {
		if m.Role == "assistant" {
			// ThinkingBlocks are skipped for Chat Completions

			if len(m.ToolUses) > 0 {
				// Assistant message with tool calls
				assistant := openai.ChatCompletionAssistantMessageParam{}
				if m.Content != "" {
					assistant.Content.OfString = param.NewOpt(m.Content)
				}
				for _, tu := range m.ToolUses {
					argsJSON, _ := json.Marshal(tu.Arguments)
					assistant.ToolCalls = append(assistant.ToolCalls, openai.ChatCompletionMessageToolCallParam{
						ID: tu.ToolUseID,
						Function: openai.ChatCompletionMessageToolCallFunctionParam{
							Name:      tu.ToolName,
							Arguments: string(argsJSON),
						},
					})
				}
				result = append(result, openai.ChatCompletionMessageParamUnion{OfAssistant: &assistant})
			} else if m.Content != "" {
				result = append(result, openai.AssistantMessage(m.Content))
			}
		} else if len(m.ToolResults) > 0 {
			// Tool results become individual tool messages
			for _, tr := range m.ToolResults {
				result = append(result, openai.ToolMessage(tr.Content, tr.ToolUseID))
			}
		} else {
			// User messages
			result = append(result, openai.UserMessage(m.Content))
		}
	}

	return result
}
