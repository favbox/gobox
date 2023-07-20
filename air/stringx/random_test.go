package stringx

import (
	"fmt"
	"testing"

	"github.com/zeromicro/go-zero/core/stringx"
)

func TestRand(t *testing.T) {
	fmt.Println(Randn(10))
	fmt.Println(stringx.Randn(10))
	fmt.Println(stringx.RandId())
}

func BenchmarkRandString(b *testing.B) {
	b.Run("fastrand/Randn", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = Randn(10)
		}
	})

	b.Run("go-zero/stringx-Randn", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = stringx.Randn(10)
		}
	})
}

func BenchmarkMultipleCoreRandString(b *testing.B) {
	b.Run("fastrand/Randn", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = Randn(10)
			}
		})
	})

	b.Run("go-zero/stringx-Randn", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = stringx.Randn(10)
			}
		})
	})
}
