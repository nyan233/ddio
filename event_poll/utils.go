package ddio

import "unsafe"


func noescape(pointer unsafe.Pointer) unsafe.Pointer {
	x := uintptr(pointer)
	return unsafe.Pointer(x ^ 0)
}
