package worldio

// #cgo LDFLAGS: -lz
import "C"

import (
	"sync"
)

var M sync.Mutex
