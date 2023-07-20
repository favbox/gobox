package fastrand

import (
	"unsafe"

	"github.com/favbox/gosky/air/internal/runtimex"
)

// Uint32 返回一个伪随机值作为 uint32。
func Uint32() uint32 {
	return runtimex.Fastrand()
}

// Uint64 返回一个伪随机值作为 uint64。
func Uint64() uint64 {
	return (uint64(runtimex.Fastrand()) << 32) | uint64(runtimex.Fastrand())
}

// Int 返回一个伪随机的非负整数作为 int。
func Int() int {
	u := uint(Int63())
	return int(u << 1 >> 1) // 如果 int == int32，则清除符号位
}

// Int31 返回一个伪随机的31位非负整数作为 int32。
func Int31() int32 {
	return int32(Uint32() & (1<<31 - 1))
}

// Int63 返回一个伪随机的63位非负整数作为 int64。
func Int63() int64 {
	return int64(Uint64() & (1<<63 - 1))
}

// Int63n 返回 [0，n）中的非负伪随机数作为 int64。
// 如果 n <= 0，它会恐慌。
func Int63n(n int64) int64 {
	if n <= 0 {
		panic("Int63n的参数无效")
	}
	if n&(n-1) == 0 { // n是2的幂，可以屏蔽
		return Int63() & (n - 1)
	}
	max := int64((1 << 63) - 1 - (1<<63)%uint64(n))
	v := Int63()
	for v > max {
		v = Int63()
	}
	return v % n
}

// Int31n 返回 [0，n）中的非负伪随机数作为 int32。
// 如果 n <= 0，它会恐慌。
func Int31n(n int32) int32 {
	if n <= 0 {
		panic("Int31n的参数无效")
	}
	v := Uint32()
	prod := uint64(v) * uint64(n)
	low := uint32(prod)
	if low < uint32(n) {
		thresh := uint32(-n) % uint32(n)
		for low < thresh {
			v = Uint32()
			prod = uint64(v) * uint64(n)
			low = uint32(prod)
		}
	}
	return int32(prod >> 32)
}

// Intn 返回 [0，n）中的非负伪随机数作为 int。
// 如果 n <= 0，它会恐慌。
func Intn(n int) int {
	// EQ
	if n <= 0 {
		panic("Intn的参数无效")
	}
	if n <= 1<<31-1 {
		return int(Int31n(int32(n)))
	}
	return int(Int63n(int64(n)))
}

func Float64() float64 {
	// EQ
	return float64(Int63n(1<<53)) / (1 << 53)
}

func Float32() float32 {
	// EQ
	return float32(Int31n(1<<24)) / (1 << 24)
}

// Uint32n 返回 [0,n) 中的伪随机数。
//
//go:nosplit
func Uint32n(n uint32) uint32 {
	// This is similar to Uint32() % n, but faster.
	// See https://lemire.me/blog/2016/06/27/a-fast-alternative-to-the-modulo-reduction/
	return uint32(uint64(Uint32()) * uint64(n) >> 32)
}

// Uint64n 返回 [0,n) 中的伪随机数。
func Uint64n(n uint64) uint64 {
	return Uint64() % n
}

// Read 生成 len(p) 个随机字节并写入 p。
// 它总是返回 len(p) 和一个 nil 错误。
// 它可以安全的同时调用。
func Read(p []byte) (int, error) {
	l := len(p)
	if l == 0 {
		return 0, nil
	}

	// Used for local XORSHIFT.
	// Xorshift paper: https://www.jstatsoft.org/article/view/v008i14/xorshift.pdf
	s0, s1 := Uint32(), Uint32()

	if l >= 4 {
		var i int
		uint32p := *(*[]uint32)(unsafe.Pointer(&p))
		for l >= 4 {
			// Local XORSHIFT.
			s1 ^= s1 << 17
			s1 = s1 ^ s0 ^ s1>>7 ^ s0>>16
			s0, s1 = s1, s0

			uint32p[i] = s0 + s1
			i++
			l -= 4
		}
	}

	if l > 0 {
		// Local XORSHIFT.
		s1 ^= s1 << 17
		s1 = s1 ^ s0 ^ s1>>7 ^ s0>>16

		r := s0 + s1
		for l > 0 {
			p[len(p)-l] = byte(r >> (l * 8))
			l--
		}
	}

	return len(p), nil
}

// Shuffle 伪随机化元素的顺序。
// n 是元素的数量。 如果 n < 0， Shuffle 会恐慌。
// swap 交换索引 i 和 j 对应的元素。
func Shuffle(n int, swap func(i, j int)) {
	if n < 0 {
		panic("Shuffle的参数无效")
	}
	// Fisher-Yates shuffle: https://en.wikipedia.org/wiki/Fisher%E2%80%93Yates_shuffle
	// Shuffle really ought not be called with n that doesn't fit in 32 bits.
	// Not only will it take a very long time, but with 2³¹! possible permutations,
	// there's no way that any PRNG can have a big enough internal state to
	// generate even a minuscule percentage of the possible permutations.
	// Nevertheless, the right API signature accepts an int n, so handle it as best we can.
	i := n - 1
	for ; i > 1<<31-1-1; i-- {
		j := int(Int63n(int64(i + 1)))
		swap(i, j)
	}
	for ; i > 0; i-- {
		j := int(Int31n(int32(i + 1)))
		swap(i, j)
	}
}

// Perm 返回半开区间 [0,n) 中的长度为 n 的伪随机排列(Permutation)切片。
func Perm(n int) []int {
	m := make([]int, n)
	for i := 1; i < n; i++ {
		j := Intn(i + 1)
		m[i] = m[j]
		m[j] = i
	}
	return m
}
