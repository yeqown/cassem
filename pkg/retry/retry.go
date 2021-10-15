package retry

import "context"

// Strategy 重试策略
type Strategy interface {
	// Do 调用逻辑并执行重试策略
	//
	// 注意ctx可能已经设置了超时时间，所以确保在重试过程中判断是否已经超时.
	Do(ctx context.Context, fn func() error) error
}
