package event_poll

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
	for i := 0; i < 100; i++ {
		n := rand.Intn(10)
		buffer,ok := pool.AllocBuffer(n)
		if !ok {
			t.Error("buffer pool allocation failed")
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
	t.Log(sorts)
	mallocView(pool)
}

func BenchmarkAlloc(b *testing.B) {
	pool := NewBufferPool(20,14)
	b.Run("BufferPoolAlloc", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_,_ = pool.AllocBuffer(1)
		}
	})
	b.Run("NativeAlloc", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = HeapAlloc()
		}
	})
}

func HeapAlloc() []byte {
	buf := make([]byte,1024 * 1024)
	return buf
}