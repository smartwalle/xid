package xid

import "context"

type Generator interface {
	Next(ctx context.Context) (uint64, error)

	NextBatch(ctx context.Context, n int) ([]uint64, error)
}

var defaultGenerator Generator

// Next 生成 ID
func Next(ctx context.Context) (uint64, error) {
	return defaultGenerator.Next(ctx)
}

// Default 获取默认生成器
func Default() Generator {
	return defaultGenerator
}

// UseGenerator 设置默认生成器
func UseGenerator(generator Generator) {
	if generator != nil {
		defaultGenerator = generator
	}
}

func init() {
	defaultGenerator, _ = New()
}
