package container

import (
	"strconv"
	"time"
)

/*
	代码思路参照《算法》第四版-2.4.4节
*/

// LittleHeap 小顶堆的实现
type LittleHeap struct {
	// 小顶堆底层二叉树中索引1处不使用
	binaryTree []TimeoutElem
	// 节点的数量
	n int
}

type TimeoutElem struct {
	TimeOut time.Duration
	Data    interface{}
}

func NewLittleHeap(maxLen int) *LittleHeap {
	return &LittleHeap{
		binaryTree: make([]TimeoutElem,maxLen + 1),
		n:          0,
	}
}

// cmp
func (h *LittleHeap) less(i, j int) bool {
	return h.binaryTree[i].TimeOut > h.binaryTree[j].TimeOut
}

// switch
func (h *LittleHeap) exch(i, j int) {
	t := h.binaryTree[i]
	h.binaryTree[i] = h.binaryTree[j]
	h.binaryTree[j] = t
}

// 从下至上的堆有序化(上浮)
func (h *LittleHeap) swim(k int) {
	for k > 1 && h.less(k/2, k) {
		h.exch(k/2, k)
		k = k / 2
	}
}

// 从上至下的堆有序化(下沉)
func (h *LittleHeap) sink(k int) {
	for 2*k <= h.n {
		j := 2 * k
		if j < h.n && h.less(j,j+1) {
			j++
		}
		if !h.less(k,j) {
			break
		}
		h.exch(k,j)
		k = j
	}
}

func (h *LittleHeap) IsEmpty() bool {
	return h.n == 0
}

func (h *LittleHeap) Size() int {
	return h.n
}

func (h *LittleHeap) Peek() TimeoutElem {
	return h.binaryTree[1]
}

func (h *LittleHeap) Insert(v TimeoutElem) {
	if h.n + 1 > len(h.binaryTree) - 1 {
		h.binaryTree = append(h.binaryTree,TimeoutElem{})
	}
	h.binaryTree[h.n + 1] = v
	h.n++
	h.swim(h.n)
}

func (h *LittleHeap) DelTop() TimeoutElem {
	max := h.binaryTree[1]
	h.exch(1,h.n)
	h.n -= 1
	h.binaryTree[h.n + 1] = TimeoutElem{}
	h.sink(1)
	h.binaryTree = h.binaryTree[:h.n + 2]
	return max
}

func (h *LittleHeap) String() string {
	level,index := 0,1
	rawIndex := 0
	str := "\n"
	for h.n >= index {
		str += strconv.Itoa(int(h.binaryTree[index].TimeOut)) + " "
		if index == rawIndex + (1 << level) {
			str += "\n"
			rawIndex = index
			level++
		}
		index++
	}
	return str
}
