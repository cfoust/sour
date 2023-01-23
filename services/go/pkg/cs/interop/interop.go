package interop

// #cgo CXXFLAGS: -std=c++03 
import "C"

import (
	"sync"
	"log"
)

var M sync.Mutex

//export Test
func Test(b *C.char) {
	log.Printf("called %s", C.GoString(b))
}
