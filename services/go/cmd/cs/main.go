package main

import (
	"github.com/cfoust/sour/pkg/cs"
)

func main() {
	cs.Execute(`
echo "hello world"
set a 2
echo (a)

texture = [
  _gocall texture $arg1 $arg2 $arg3 $arg4 $arg5 $arg6 $arg7 $arg8
]
texture 0 "blah.png"
texture 0 "blah2.png"
`)
}
