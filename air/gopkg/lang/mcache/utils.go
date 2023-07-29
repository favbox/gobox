package mcache

import "math/bits"

// 判断 x 是否为2的倍数。性能略快于求商取余。
func isPowerOfTwo(x int) bool {
	return (x & (-x)) == x
}

// 返回 x 的二进制表示所需的最小位数。
//
// Bit Scan Reverse 用于查找给定整数中从最高有效位（最左边的非零位）
// 开始的第一个置位（bit set）位的位置索引。
func bsr(x int) int {
	return bits.Len(uint(x)) - 1
}
