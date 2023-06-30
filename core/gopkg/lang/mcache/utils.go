package mcache

import "math/bits"

// 判断 x 是否为2的倍数。性能略快于求模取余。
func isPowerOfTwo(x int) bool {
	return (x & (-x)) == x
}

// 返回表示 x 所需的最小位数。
func bsr(x int) int {
	return bits.Len(uint(x)) - 1
}
