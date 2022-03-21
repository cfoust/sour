package main

import (
	"fmt"
	"github.com/cfoust/sour/pkg/enet"
)

func main() {
	socket, err := enet.NewSocket("master.sauerbraten.org", 28785)
	if err != nil {
		fmt.Println("Error creating socket")
	}
	fmt.Println("OK")
	socket.DestroySocket()
}
