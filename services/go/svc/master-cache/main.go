package main

import (
	"fmt"
	"github.com/cfoust/sour/pkg/enet"
)

func main() {
	socket, err := enet.NewSocket("master.sauerbraten.org", 28787)
	if err != nil {
		fmt.Println("Error creating socket")
	}
	fmt.Println("OK")
	socket.SendString("list\n")
	val, length := socket.Receive()
	if length < 0 {
		fmt.Println("Error fetching server list")
	}
	fmt.Println(val)
	socket.DestroySocket()
}
