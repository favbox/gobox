package bytebufferpool

import (
	"sort"
	"sync"
	"sync/atomic"
)

const (
	minBitSize = 6 // 2**6=64 是CPU缓存行大小
	steps      = 20

	minSize = 1 << minBitSize
	maxSize = 1 << (minBitSize + steps - 1)

	calibrateCallsThreshold = 42000
	maxPercentile           = 0.95
)

// Pool 表示字节缓冲池。
//
// 不同的吃可用于不同类型的字节缓冲区。
type Pool struct {
	calls       [steps]uint64
	calibrating uint64 // 校准值

	defaultSize uint64
	maxSize     uint64

	pool sync.Pool
}

var defaultPool Pool

// Get 返回缓冲池中的一个空缓冲区。
func Get() *ByteBuffer { return defaultPool.Get() }

// Get 返回一个长度为零的新字节缓冲区。
//
// 字节缓冲区可在用后通过 Put 放回池中，以最大限度减少 GC 开销。
func (p *Pool) Get() *ByteBuffer {
	v := p.pool.Get()
	if v != nil {
		return v.(*ByteBuffer)
	}
	return &ByteBuffer{
		B: make([]byte, 0, atomic.LoadUint64(&p.defaultSize)),
	}
}

// Put 将字节缓冲区放回池中。
//
// ByteBuffer.B 放回池中以后不可再触碰，否则将引发数据竞赛。
func Put(b *ByteBuffer) { defaultPool.Put(b) }

// Put 将通过 Get 获取的字节缓冲区放入池中。
func (p *Pool) Put(b *ByteBuffer) {
	idx := index(len(b.B))

	if atomic.AddUint64(&p.calls[idx], 1) > calibrateCallsThreshold {
		p.calibrate()
	}

	maxSize := int(atomic.LoadUint64(&p.maxSize))
	if maxSize == 0 || cap(b.B) <= maxSize {
		b.Reset()
		p.pool.Put(b)
	}
}

func (p *Pool) calibrate() {
	if !atomic.CompareAndSwapUint64(&p.calibrating, 0, 1) {
		return
	}

	a := make(callSizes, 0, steps)
	var callsSum uint64
	for i := uint64(0); i < steps; i++ {
		calls := atomic.SwapUint64(&p.calls[i], 0)
		callsSum += calls
		a = append(a, callSize{
			calls: calls,
			size:  minSize << i,
		})
	}
	sort.Sort(a)

	defaultSize := a[0].size
	maxSize := defaultSize

	maxSum := uint64(float64(callsSum) * maxPercentile)
	callsSum = 0
	for i := 0; i < steps; i++ {
		if callsSum > maxSum {
			break
		}
		callsSum += a[i].calls
		size := a[i].size
		if size > maxSize {
			maxSize = size
		}
	}

	atomic.StoreUint64(&p.defaultSize, defaultSize)
	atomic.StoreUint64(&p.maxSize, maxSize)

	atomic.StoreUint64(&p.calibrating, 0)
}

type callSize struct {
	calls uint64
	size  uint64
}

type callSizes []callSize

func (cs callSizes) Len() int {
	return len(cs)
}

func (cs callSizes) Less(i, j int) bool {
	return cs[i].calls > cs[j].calls
}

func (cs callSizes) Swap(i, j int) {
	cs[i], cs[j] = cs[j], cs[i]
}

func index(n int) int {
	n--
	n >>= minBitSize
	idx := 0
	for n > 0 {
		n >>= 1
		idx++
	}
	if idx >= steps {
		idx = steps - 1
	}
	return idx
}
