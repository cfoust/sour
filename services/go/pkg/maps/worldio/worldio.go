package worldio

// #cgo LDFLAGS: -lz
import "C"

import (
	"sync"
)

var M sync.Mutex

var refMutex sync.Mutex
var refs = make([]int, 0)

//export Ref
func Ref(index int) {
	refs = append(refs, index)
}

func CountRefs(state MapState) []int {
	refMutex.Lock()
	defer refMutex.Unlock()
	Getrefs(state)
	return refs
}
