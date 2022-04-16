package ddio

import (
	"github.com/zbh255/bilog"
	"os"
)

var (
	logger bilog.Logger = bilog.NewLogger(os.Stdout,bilog.PANIC,bilog.WithTimes(),bilog.WithCaller())
)
