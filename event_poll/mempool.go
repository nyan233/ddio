package ddio

import (
	"fmt"
	"math"
	"reflect"
	"sync"
	"unsafe"
)

// 负责管理buffer的内存池

const (
	DEFAULT_BLOCK              = 4096                 // 一个内存块的默认大小
	DEFAULT_POOL_SIZE          = DEFAULT_BLOCK * 8192 // 默认池大小 (DEFAULT_BLOCK * 8192) B
	FreeBufferZeroBase uintptr = math.MaxInt
	maxUint64          uint64  = 0xffffffffffffffff
	sysPageSize        int     = 4096
	defaultPending     byte    = 253
)

// NewBufferPool
// block 是块大小，会转换为2的N次方
// size 是池能容纳的块数量,会转换为2的N次方
// -1则使用默认配置
func NewBufferPool(block, size int) *BufferPool {
	if block >= 64 || size >= 64 {
		return nil
	} else if block == -1 || size == -1 {
		block = DEFAULT_BLOCK
		size = DEFAULT_POOL_SIZE
	} else {
		block = 1 << block
		size = (1 << size) * block
	}
	heapSlice := make([]byte, size)

	return &BufferPool{
		block:    int32(block),
		size:     int32(size),
		pool:     &heapSlice,
		allocMap: make([]uint64, size/block/64),
	}
}

type BufferPool struct {
	mu sync.Mutex
	// 初始化的Buffer内存池
	// *[]byte使它更容易被gc回收
	pool *[]byte
	// 记录分配状况的BitMap
	allocMap []uint64
	// block
	block int32
	// pool size
	size int32
}

// AllocBuffer variable == n * 256
func (p *BufferPool) AllocBuffer(n int) ([]byte, bool) {
	if !checkN(n) {
		return nil, false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	// 一次只支持最多分配一个BitMap的大小，即: 512*64 B
	if (1<<n)-1 == maxUint64 {
		return nil, false
	}
	poolPtr := (*reflect.SliceHeader)(unsafe.Pointer(p.pool)).Data
	// slice ptr and alloc ok
	var ptr uintptr
	var allocOk bool
	// 寻找可以分配的span
	for i := 0; i < len(p.allocMap); i++ {
		// 重置内存池的临时指针
		ptr = poolPtr + uintptr(64*int(p.block)*i)
		// 没有分配过的情况则重头开始分配
		if p.allocMap[i]^maxUint64 == maxUint64 {
			allocOk = true
			// 标记分配情况
			var allocNSpan uint64 = (1 << n) - 1
			p.allocMap[i] |= allocNSpan
			break
		}
		// 寻找合适的位置为其分配
		// 如果找不到合适的空间则分配失败
		// 分配全满则遍历剩下的BitMap
		if ^p.allocMap[i] == 0 {
			continue
		}
		for j := 1; j <= 64-n; j++ {
			// N == 4
			// AllocMap : 11110000011
			// Sa : (1 << 4) - 1 = 01111
			// SaN : Sa << J = 01111 << 1 = 11110
			// XorMap : AllocMap ^ SaN = 11110000011 ^ 00000011110 = 11110011101
			// Bl : XorMap & SaN == SaN = 11110011101 & 00000011110 = 00000011100 == 00000011110 (False)
			var sa uint64 = (1 << n) - 1
			var saN uint64 = sa << j
			tBitMap := p.allocMap[i]
			xorTmp := tBitMap ^ saN
			// 有空余的位置，分配成功
			if xorTmp&saN == saN {
				p.allocMap[i] |= saN
				ptr += uintptr((j - 1) * int(p.block))
				allocOk = true
				break
			}
		}
		// check ok
		if allocOk {
			break
		}
	}

	header := &reflect.SliceHeader{
		Data: ptr,
		Len:  0,
		Cap:  int(p.block) * n,
	}
	return *(*[]byte)(unsafe.Pointer(header)), allocOk
}

// FreeBuffer 释放分配出去的Buffer内存
func (p *BufferPool) FreeBuffer(ptr *[]byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	header := (*reflect.SliceHeader)(unsafe.Pointer(ptr))
	poolPtr := (*reflect.SliceHeader)(unsafe.Pointer(p.pool)).Data
	offset := header.Data - poolPtr
	// 判断该Buffer是否由MemPool分配
	if offset%uintptr(p.block) != 0 || offset > uintptr(p.size) {
		panic("the buffer is not BufferPool alloc")
	}
	// 重置原来的指针使其指向runtime.ZeroBase
	header.Data = FreeBufferZeroBase
	// 判断由哪一个bitMap记录其分配情况
	nBitMap := offset / uintptr(p.block) / 64
	// 判断被分配在哪个块了
	iBlock := (offset / uintptr(p.block)) % 64
	// 使用了多少个块分配
	nBlock := header.Cap / int(p.block)
	// NBitMap == 4, IBlock == 7, NBlock == 8
	// FreeAllocMap : ((1 << nBlock) - 1) << iBlock = 11111111 << 7 = 111111110000000
	var freeAllocMap uint64 = ((1 << nBlock) - 1) << iBlock
	p.allocMap[nBitMap] ^= freeAllocMap
}

// Grow 扩容原来的Buffer,可指定的扩容大小为n * p.block
// Example
//	pool.Grow(&buffer,4)
func (p *BufferPool) Grow(ptr *[]byte, nBlock int) bool {
	if !checkN(nBlock) {
		return false
	}
	p.mu.Lock()
	header := (*reflect.SliceHeader)(unsafe.Pointer(ptr))
	// 计算扩容Buffer需要多少Block
	growNBlock := header.Cap/int(p.block) + nBlock
	p.mu.Unlock()
	growBuffer, ok := p.AllocBuffer(growNBlock)
	if !ok {
		return ok
	}
	// 分配新的Buffer并将旧的Buffer拷贝进去
	growBuffer = growBuffer[:header.Len]
	copy(growBuffer, *ptr)
	// 释放旧的Buffer
	p.FreeBuffer(ptr)
	return true
}

// 打印分配的情况
func mallocView(ptr *BufferPool) {
	fmt.Printf("BufferPool : Size : %d, Pointer : %p, Block : %d\n", ptr.size, ptr.pool, ptr.block)
	for k, v := range ptr.allocMap {
		fmt.Printf("(%d) -> %b\n", k+1, v)
	}
}

func checkN(n int) bool {
	return n > 0
}
