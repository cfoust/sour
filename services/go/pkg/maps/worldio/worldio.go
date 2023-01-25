package worldio

// #cgo LDFLAGS: -lz
import "C"

import (
	"sync"
	"unsafe"

	"github.com/cfoust/sour/pkg/game"
)

var M sync.Mutex

func CountRefs(state MapState, numSlots int) []int32 {
	data := make([]byte, numSlots * 4)
	Getrefs(state, uintptr(unsafe.Pointer(&data[0])), numSlots)

	result := make([]int32, 0)
	buffer := game.Buffer(data)
	for i := 0; i < numSlots; i++ {
		value, ok := buffer.GetInt()
		if !ok {
			return result
		}
		result = append(result, value)
	}
	return result
}
