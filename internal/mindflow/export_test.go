package mindflow

import "context"

// RunVariantForTest advances one preview job without waiting for the worker timer.
func (a *App) RunVariantForTest(ctx context.Context) { a.images.RunVariant(ctx) }
