package ddio

type RoundBalanced struct {
	connNumber int64
}

func (r *RoundBalanced) Name() string {
	return "default-round"
}

func (r *RoundBalanced) Target(connLen, fd int) int {
	if int(r.connNumber) >= connLen {
		r.connNumber = 0
	}
	r.connNumber++
	return int(r.connNumber - 1)
}
