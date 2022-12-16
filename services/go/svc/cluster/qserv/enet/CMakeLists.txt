cmake_minimum_required(VERSION 2.8)
project(enet)

set(enet_srcs
	host.c
	list.c
	callbacks.c
	packet.c
	peer.c
	protocol.c
	unix.c
	win32.c)


add_definitions(-DHAS_SOCKLEN_T)

include_directories(include)

add_library(enet STATIC ${enet_srcs})
