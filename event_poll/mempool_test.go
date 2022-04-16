package ddio

import (
	"math/rand"
	"reflect"
	"sort"
	"testing"
	"time"
	"unsafe"
)

func TestMemPool(t *testing.T) {
	pool := NewBufferPool(-1,-1)
	rand.Seed(time.Now().UnixNano())
	str := "hello world"
	var sliceCollections []*reflect.SliceHeader
	for i := 0; i < 2000; i++ {
		n := rand.Intn(10)
		buffer,ok := pool.AllocBuffer(n)
		if !ok {
			continue
		}
		buffer = append(buffer,str...)
		sliceCollections = append(sliceCollections,(*reflect.SliceHeader)(unsafe.Pointer(&buffer)))
	}
	var sorts []int
	// 检查分配的内存地址是否有冲突
	for _,v := range sliceCollections {
		sorts = append(sorts,int(v.Data))
	}
	sort.Ints(sorts)
	// random free
	for _,v := range sliceCollections {
		buf := (*[]byte)(unsafe.Pointer(v))
		if int32(cap(*buf)) / pool.block % 2 == 0 {
			pool.FreeBuffer(buf)
		}
	}
	mallocView(pool)
}

func BenchmarkAlloc(b *testing.B) {

	b.Run("BigBufferPoolAlloc", func(b *testing.B) {
		pool := NewBufferPool(20,10)
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf,ok := pool.AllocBuffer(1)
			if !ok {
				continue
			}
			if i % 2 == 0 {
				pool.FreeBuffer(&buf)
			}
		}
	})
	b.Run("BigBufferNativeAlloc", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := HeapAlloc(1 << 20)
			FreeAlloc(&buf)
		}
	})
}

func HeapAlloc(n int) []byte {
	buf := make([]byte,n)
	return buf
}

func FreeAlloc(ptr *[]byte) {
	header := (*reflect.SliceHeader)(unsafe.Pointer(ptr))
	header.Data = 0
	header.Len = 0
	header.Cap = 0
}