package ddio

import "sync/atomic"

type RoundBalanced struct {
	connNumber int64
}

func (r *RoundBalanced) Name() string {
	return "default-round"
}

func (r *RoundBalanced) Target(seek int) int {
	atomic.AddInt64(&r.connNumber,1)
	return int(atomic.LoadInt64(&r.connNumber)) % seek
}

