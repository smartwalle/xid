package xid

import (
	"context"
	"testing"
)

func BenchmarkGeneratorNext(b *testing.B) {
	g, err := New()
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err = g.Next(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPackageNext(b *testing.B) {
	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := Next(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGeneratorNextBatch(b *testing.B) {
	const batchSize = 100

	g, err := New(WithMaxBatchSize(batchSize))
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err = g.NextBatch(ctx, batchSize); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGeneratorNextParallel(b *testing.B) {
	g, err := New()
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := g.Next(ctx); err != nil {
				b.Fatal(err)
			}
		}
	})
}
