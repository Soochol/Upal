package model

import "context"

// LogFunc is a callback for emitting execution-level log messages.
// LLM implementations call this to surface internal details (CLI args,
// timings, errors) without coupling to any event or transport system.
type LogFunc func(message string)

type logFuncKey struct{}

// WithLogFunc returns a context carrying a log callback.
func WithLogFunc(ctx context.Context, fn LogFunc) context.Context {
	return context.WithValue(ctx, logFuncKey{}, fn)
}

// emitLog calls the log callback on the context, if one is set.
// Safe to call when no callback is registered (no-op).
func emitLog(ctx context.Context, msg string) {
	if fn, ok := ctx.Value(logFuncKey{}).(LogFunc); ok {
		fn(msg)
	}
}
