package api

import (
	"context"
	"errors"
)

type CRI interface {
	List(ctx context.Context, podUID string) ([]string, error)
	Pause(ctx context.Context, container string) error
	Resume(ctx context.Context, container string) error
}

var ErrNoNonQueueProxyPods = errors.New("no non queue-proxy containers found in pod")