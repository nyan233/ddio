package ddio

import "testing"

//TODO:  ConnMultiEventDispatcher的逃逸分析视图
func TestConnMultiEventDispatcherEscape(t *testing.T) {
	//defer func() {
	//	err := recover()
	//	t.Error(err)
	//}()
	_, _ = NewConnMultiEventDispatcher(nil, DefaultConfig)
}
