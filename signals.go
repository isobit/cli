package cli

import (
	"context"
	"os/signal"
	"syscall"
)

func ContextWithSigCancel(ctx context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
}
