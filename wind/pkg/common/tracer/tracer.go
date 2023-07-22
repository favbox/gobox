package tracer

import (
	"context"

	"github.com/favbox/gosky/wind/pkg/app"
)

// Tracer 在 HTTP 开始和结束时执行。
type Tracer interface {
	Start(ctx context.Context, c *app.RequestContext) context.Context
	Finish(ctx context.Context, c *app.RequestContext)
}

// Controller 跟踪控制器
type Controller interface {
	Append(col Tracer)
	DoStart(ctx context.Context, c *app.RequestContext) context.Context
	DoFinish(ctx context.Context, c *app.RequestContext, err error)
	HasTracer() bool
}
