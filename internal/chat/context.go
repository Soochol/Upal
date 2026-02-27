package chat

import "context"

type chatContextKey struct{}

func WithChatContext(ctx context.Context, chatCtx map[string]any) context.Context {
	return context.WithValue(ctx, chatContextKey{}, chatCtx)
}

func GetChatContext(ctx context.Context) map[string]any {
	if v, ok := ctx.Value(chatContextKey{}).(map[string]any); ok {
		return v
	}
	return nil
}
