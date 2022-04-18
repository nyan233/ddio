package ddio

import "sync/atomic"

type RoundBalanced struct {
	connNumber int64
}

func (r *RoundBalanced) Name() string {
	return "default-round"
}

func (r *RoundBalanced) Target(connLen, fd int) int {
	atomic.AddInt64(&r.connNumber, 1)
	return fd % connLen
}
