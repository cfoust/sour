package main

import (
	"log"
	"github.com/cfoust/sour/pkg/cs"
)

func Test(a int) {
	log.Printf("%d", a)
}

func MdlName() string {
	return "mdl"
}

func main() {
	vm := cs.NewVM()

	vm.AddCommand("test", Test)
	vm.AddCommand("mdlname", MdlName)
	vm.Run(`
echo (mdlname)
`)
}
