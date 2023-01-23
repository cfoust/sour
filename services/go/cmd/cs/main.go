package main

import (
	"github.com/cfoust/sour/pkg/cs/interop"
)

func main() {
	interop.Execute(`
echo "hello world"
set a 2
echo (a)
mdlname
`)
}
