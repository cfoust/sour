package main

import (
	"github.com/cfoust/sour/pkg/cs/interop"
)

func main() {
	interop.Execute(`
set a 2
echo (a)
`)
}
