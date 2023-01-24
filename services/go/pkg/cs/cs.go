package cs

// #cgo CXXFLAGS: -std=c++03 
import "C"

import (
	"sync"
	"log"
)

var M sync.Mutex

//export Test
func Test() {
	log.Printf("called")
}
