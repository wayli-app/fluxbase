package ai

import (
	"context"
	"fmt"
	"sync"
)

// fakeProvider is a test Provider that returns canned responses. It is
// defined here (in the ai package) so all *_test.go files in this package
// can use it without re-declaring.
type fakeProvider struct {
	name      string
	mu        sync.Mutex
	responses []*ChatResponse
	errors    []error
	calls     int
	streamFn  func(callback StreamCallback) error
	chatFn    func(req *ChatRequest) (*ChatResponse, error)
}

func newFakeProvider(name string) *fakeProvider {
	return &fakeProvider{name: name}
}

func (p *fakeProvider) Name() string          { return p.name }
func (p *fakeProvider) Type() ProviderType    { return ProviderTypeOpenAI }
func (p *fakeProvider) ValidateConfig() error { return nil }
func (p *fakeProvider) Close() error          { return nil }

// QueueResponse queues a canned response for the next Chat() call.
func (p *fakeProvider) QueueResponse(resp *ChatResponse) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.responses = append(p.responses, resp)
}

// QueueError queues an error for the next Chat() call.
func (p *fakeProvider) QueueError(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.errors = append(p.errors, err)
}

// SetChatFn overrides the default Chat() behavior with a custom function.
// Use this when the canned response queue isn't flexible enough.
func (p *fakeProvider) SetChatFn(fn func(req *ChatRequest) (*ChatResponse, error)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.chatFn = fn
}

// Calls returns the number of Chat() invocations.
func (p *fakeProvider) Calls() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

// Chat returns the next queued response or error.
func (p *fakeProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	p.mu.Lock()
	p.calls++
	chatFn := p.chatFn
	var nextResp *ChatResponse
	var nextErr error
	if len(p.responses) > 0 {
		nextResp = p.responses[0]
		p.responses = p.responses[1:]
	}
	if len(p.errors) > 0 {
		nextErr = p.errors[0]
		p.errors = p.errors[1:]
	}
	p.mu.Unlock()

	if chatFn != nil {
		return chatFn(req)
	}
	if nextErr != nil {
		return nil, nextErr
	}
	if nextResp != nil {
		return nextResp, nil
	}
	return nil, fmt.Errorf("fakeProvider: no queued response")
}

// ChatStream is unused in supervisor tests — falls back to error.
func (p *fakeProvider) ChatStream(ctx context.Context, req *ChatRequest, callback StreamCallback) error {
	if p.streamFn != nil {
		return p.streamFn(callback)
	}
	return fmt.Errorf("fakeProvider: ChatStream not configured")
}

// Compile-time assertion.
var _ Provider = (*fakeProvider)(nil)

// newSimpleResponse builds a ChatResponse with one text choice.
func newSimpleResponse(text string) *ChatResponse {
	return &ChatResponse{
		ID:      "resp-test",
		Model:   "test-model",
		Choices: []Choice{{Index: 0, Message: Message{Role: RoleAssistant, Content: text}, FinishReason: "stop"}},
		Usage:   &UsageStats{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
	}
}
