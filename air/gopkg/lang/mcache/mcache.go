package mcache

import (
	"sync"
)

// 定义有几个缓存池
const maxSize = 46

// 一组缓存池。每个池的切片容量为 1<<index
var caches [maxSize]sync.Pool

// 初始化缓存切片中的每个缓存池
func init() {
	for i := 0; i < maxSize; i++ {
		size := 1 << i
		caches[i].New = func() any {
			buf := make([]byte, 0, size)
			return buf
		}
	}
}

// 计算从哪个缓存池中获取
func calcIndex(size int) int {
	if size == 0 {
		return 0
	}
	if isPowerOfTwo(size) {
		return bsr(size)
	}
	return bsr(size) + 1
}

// Malloc 利用缓存池高效分配内存。
//
// 必选参数 size 指定切片的长度，故 len(ret) == size。
// 可选参数 capacity 指定最小容量，即 cap(ret) >= capacity。
func Malloc(size int, capacity ...int) []byte {
	if len(capacity) > 1 {
		panic("Malloc 的参数只能为1或2个")
	}

	// 确定申请的内存容量
	var c = size
	if len(capacity) > 0 && capacity[0] > size {
		c = capacity[0]
	}

	// 计算该容量对应的缓存池，并从中拿出任意一项
	var ret = caches[calcIndex(c)].Get().([]byte)

	// 填充为长度为 size 的零值空切片
	ret = ret[:size]
	return ret
}

// Free 当不再使用 buf 时，应释放并放回池中。
func Free(buf []byte) {
	size := cap(buf)

	// 跳过容量为奇数的字节切片（Malloc 创建的 size 都是 2^n，即偶数）
	if !isPowerOfTwo(size) {
		return
	}

	// 处理容量为偶数的字节切片
	// 清空后放回池中等待下次使用
	buf = buf[:0]
	caches[bsr(size)].Put(buf)
}
