package interop

// #cgo CXXFLAGS: -std=c++03 
import "C"

import (
	"sync"
)

var M sync.Mutex
