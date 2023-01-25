package worldio

// #cgo LDFLAGS: -lz
import "C"

import (
	"sync"
	"unsafe"
)

var M sync.Mutex

func CountRefs(state MapState, numSlots int) []int32 {
	result := make([]int32, numSlots)
	Getrefs(state, uintptr(unsafe.Pointer(&result[0])), numSlots)
	return result
}
