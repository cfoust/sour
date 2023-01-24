package cs

// #cgo CXXFLAGS: -std=c++03
import "C"

import (
	"log"
	"sync"
)

var M sync.Mutex

//export GoCall
func GoCall(
	name *C.char,
	_1 *C.char,
	_2 *C.char,
	_3 *C.char,
	_4 *C.char,
	_5 *C.char,
	_6 *C.char,
	_7 *C.char,
	_8 *C.char,
	_9 *C.char,
	_10 *C.char,
	_11 *C.char,
	_12 *C.char,
) {
	log.Printf("called %s", C.GoString(name))
}
