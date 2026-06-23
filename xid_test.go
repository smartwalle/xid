package xid_test

import (
	"context"
	"testing"

	"github.com/smartwalle/xid"
)

func TestXID_Next(t *testing.T) {
	for i := 0; i < 100; i++ {
		t.Log(xid.Next(context.Background()))
	}
}

func BenchmarkGeneratorNext(b *testing.B) {
	g, err := xid.New()
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
		if _, err := xid.Next(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGeneratorNextBatch(b *testing.B) {
	const batchSize = 100

	g, err := xid.New(xid.WithMaxBatchSize(batchSize))
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
	g, err := xid.New()
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, nErr := g.Next(ctx); nErr != nil {
				b.Fatal(nErr)
			}
		}
	})
}
