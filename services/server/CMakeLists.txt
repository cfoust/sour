cmake_minimum_required(VERSION 2.7)

SET(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -std=c++11")
set(CMAKE_CXX_STANDARD 11)


project(qserv)
subdirs(enet GeoIP)
find_package(ZLIB REQUIRED)
find_package(Threads REQUIRED)
set(qserv_srcs
	engine/server.cpp
	engine/command.cpp
	fpsgame/server.cpp
	mod/QServ.cpp
	mod/QCom.cpp
	shared/tools.cpp
	shared/stream.cpp
	shared/crypto.cpp
	ircbot/ircbot.cpp
	)

set(qserv_LIBS
	enet
	GeoIP
	${ZLIB_LIBRARIES}
	${CMAKE_THREAD_LIBS_INIT})


IF(WIN32)
set(qserv_LIBS ${qserv_LIBS} GeoIP)
ENDIF(WIN32)

include(CheckFunctionExists)
include(CheckSymbolExists)

IF(CMAKE_COMPILER_IS_GNUCXX)
set(CMAKE_CXX_FLAGS "-rdynamic")

set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -include mod/HTTPRequest.hpp")

ENDIF(CMAKE_COMPILER_IS_GNUCXX)

check_function_exists(gethostbyaddr_r, HAS_GETHOSTBYADDR_R)
check_function_exists(gethostbyname_r, HAS_GETHOSTBYNAME_R)
check_function_exists(poll, HAS_POLL)
check_function_exists(fcntl, HAS_FCNTL)
check_function_exists(inet_pton, HAS_INET_PTON)
check_function_exists(inet_ntop, HAS_INET_NTOP)

check_symbol_exists("struct msghdr.msg_flags" "sys/socket.h" HAS_MSGHDR_FLAGS)
check_symbol_exists("socklen_t" "sys/types.h;sys/socket.h" HAS_SOCKLEN_T)

add_definitions(-DSTANDALONE)

include_directories(${CMAKE_SOURCE_DIR}/src/engine
	${CMAKE_SOURCE_DIR}/fpsgame
	${CMAKE_SOURCE_DIR}/GeoIP/libGeoIP
	${CMAKE_SOURCE_DIR}/shared
	${CMAKE_SOURCE_DIR}/tools
	${CMAKE_SOURCE_DIR}/QServ
	${CMAKE_SOURCE_DIR}/enet/include
	${ZLIB_INCLUDE_PATH})

add_executable(qserv
	${qserv_srcs})

target_link_libraries(qserv ${qserv_LIBS})

install(TARGETS qserv DESTINATION bin)

