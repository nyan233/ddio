package ddio

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"
	"unsafe"
)

func TestMemPool(t *testing.T) {
	pool := NewBufferPool(-1, -1)
	rand.Seed(time.Now().UnixNano())
	str := "hello world"
	var sliceCollections []*reflect.SliceHeader
	for i := 0; i < 2000; i++ {
		n := rand.Intn(10)
		buffer, ok := pool.AllocBuffer(n)
		if !ok {
			continue
		}
		buffer = append(buffer, str...)
		sliceCollections = append(sliceCollections, (*reflect.SliceHeader)(unsafe.Pointer(&buffer)))
	}
	var sorts []int
	// 检查分配的内存地址是否有冲突
	for _, v := range sliceCollections {
		sorts = append(sorts, int(v.Data))
	}
	sort.Ints(sorts)
	// random free
	for _, v := range sliceCollections {
		buf := (*[]byte)(unsafe.Pointer(v))
		if int32(cap(*buf))/pool.block%2 == 0 {
			pool.FreeBuffer(buf)
		}
	}
}

// 测试MemoryPool每个方法的功能
func TestFunction(t *testing.T) {
	pool := NewBufferPool(13, 10)
	buf, ok := pool.AllocBuffer(1)
	if !ok {
		t.Fatal(errors.New("alloc buffer failed"))
	}
	if pool.BlockSize() != 1<<13 {
		t.Fatal(errors.New("pool block size is error"))
	}
	if pool.Size() != (1<<10)*(1<<13) {
		t.Fatal(errors.New("pool size is error"))
	}
	ok = pool.Grow(&buf, 2)
	if !ok {
		t.Fatal(errors.New("pool grow failed"))
	}
	if cap(buf) != pool.BlockSize()*2 {
		t.Fatal(errors.New("pool grow size not equal to 2 block size"))
	}
	if !pool.IsAlloc(buf) {
		t.Fatal(errors.New("is pool alloc but IsAlloc return false"))
	}
	pool.FreeBuffer(&buf)
}

func BenchmarkAlloc(b *testing.B) {
	for i := 0; i <= 5; i++ {
		nBlock := 1 << (8 + i)
		runStr := fmt.Sprintf("4096B-%dBlock-MemPoolAlloc", nBlock)
		b.Run(runStr, func(b *testing.B) {
			pool := NewBufferPool(12, 8+i)
			b.ReportAllocs()
			b.StartTimer()
			rand.Seed(time.Now().UnixNano())
			for i := 0; i < b.N; i++ {
				buf, ok := pool.AllocBuffer(1)
				if !ok {
					continue
				}
				if rand.Intn(10)+1 > 5 {
					pool.FreeBuffer(&buf)
				}
			}
			b.StopTimer()
		})
	}
	b.Run("4096B-MemPoolAllocAll", func(b *testing.B) {
		pool := NewBufferPool(12, 8)
		b.ReportAllocs()
		b.StartTimer()
		for i := 0; i < b.N; i++ {
			MemPollAllocAll(pool, 1<<8)
			MemPoolFreeAll(pool, 1<<8)
		}
		b.StopTimer()
	})
	b.Run("4096B-NativeAlloc", func(b *testing.B) {
		b.ReportAllocs()
		b.StartTimer()
		rand.Seed(time.Now().UnixNano())
		for i := 0; i < b.N; i++ {
			buf := HeapAlloc(1 << 12)
			if rand.Intn(10)+1 > 5 {
				FreeAlloc(&buf)
			}
		}
		b.StopTimer()
	})
	b.Run("4096B-NativeAllocAll", func(b *testing.B) {
		b.ReportAllocs()
		b.StartTimer()
		const allocN = 1 << 8
		allocMap := [allocN][]byte{}
		for i := 0; i < b.N; i++ {
			for i := 0; i < allocN; i++ {
				allocMap[i] = HeapAlloc(4096)
			}
			for i := 0; i < allocN; i++ {
				FreeAlloc(&allocMap[i])
			}
		}
	})
	b.Run("4096B-NativeStackAlloc", func(b *testing.B) {
		b.ReportAllocs()
		b.StartTimer()
		for i := 0; i < b.N; i++ {
			_ = make([]byte, 4096)
		}
		b.StopTimer()
	})
	b.Run("4096B-SyncPoolAlloc", func(b *testing.B) {
		b.ReportAllocs()
		pool := sync.Pool{
			New: func() interface{} {
				return make([]byte, 4096)
			},
		}
		rand.Seed(time.Now().UnixNano())
		for i := 0; i < b.N; i++ {
			buf := pool.Get().([]byte)
			if rand.Intn(10)+1 > 5 {
				pool.Put(buf)
			}
		}
	})
}

// 测试MemoryPool每个方法的性能
func BenchmarkFn(b *testing.B) {
	b.Run("IsAlloc", func(b *testing.B) {
		b.ReportAllocs()
		memPool := NewBufferPool(10, 10)
		buf, _ := memPool.AllocBuffer(1)
		for i := 0; i < b.N; i++ {
			_ = memPool.IsAlloc(buf)
		}
	})
}

func HeapAlloc(n int) []byte {
	buf := make([]byte, n)
	return buf
}

func FreeAlloc(ptr *[]byte) {
	header := (*reflect.SliceHeader)(unsafe.Pointer(ptr))
	header.Data = 0
	header.Len = 0
	header.Cap = 0
}

func MemPollAllocAll(pool *MemoryPool, allocN int) {
	for i := 0; i < allocN; i++ {
		pool.AllocBuffer(1)
	}
}

func MemPoolFreeAll(pool *MemoryPool, allocN int) {
	for i := 0; i < allocN; i++ {
		slice := &reflect.SliceHeader{
			Data: (*reflect.SliceHeader)(unsafe.Pointer(pool.pool)).Data + uintptr(i*int(pool.block)),
			Len:  0,
			Cap:  int(pool.block),
		}
		pool.FreeBuffer((*[]byte)(unsafe.Pointer(slice)))
	}
}
