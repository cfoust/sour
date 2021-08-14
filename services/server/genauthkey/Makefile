CXXFLAGS= -O2 -g -pipe -fstack-protector
override CXXFLAGS+= -Wall -fsigned-char -fno-exceptions -fno-rtti

PLATFORM= $(shell uname -s)
PLATFORM_PREFIX= native

INCLUDES= -I../shared -I../enet/include

STRIP=
ifeq (,$(findstring -g,$(CXXFLAGS)))
ifeq (,$(findstring -pg,$(CXXFLAGS)))
  STRIP=strip
endif
endif

MV=mv

ifneq (,$(findstring MINGW,$(PLATFORM)))
ifneq (,$(findstring 64,$(PLATFORM)))
ifneq (,$(findstring CROSS,$(PLATFORM)))
  CXX=x86_64-w64-mingw32-g++
ifneq (,$(STRIP))
  STRIP=x86_64-w64-mingw32-strip
endif
endif
WINLIB=../lib64
WINBIN=../../bin64
override CXX+= -m64
else
ifneq (,$(findstring CROSS,$(PLATFORM)))
  CXX=i686-w64-mingw32-g++
ifneq (,$(STRIP))
  STRIP=i686-w64-mingw32-strip
endif
endif
WINLIB=../lib
WINBIN=../../bin
override CXX+= -m32
endif
ifneq (,$(findstring TDM,$(PLATFORM)))
STD_LIBS=
else
STD_LIBS= -static-libgcc -static-libstdc++
endif
endif

ifneq (,$(findstring MINGW,$(PLATFORM)))
GAK_INCLUDES= -DSTANDALONE $(INCLUDES) -Iinclude
GAK_LIBS= $(STD_LIBS) -L$(WINBIN) -L$(WINLIB) -lzlib1 -lenet -lws2_32 -lwinmm
else
GAK_INCLUDES= -DSTANDALONE $(INCLUDES)
GAK_LIBS= -L../enet -lenet -lz
endif

GAK_OBJS= \
	../shared/tools-standalone.o \
	../shared/stream-standalone.o \
	genauthkey-standalone.o

default: all

all: genauthkey

clean:
	-$(RM) $(GAK_OBJS) genauthkey


%-standalone.o: %.cpp
	$(CXX) $(CXXFLAGS) -c -o $@ $(subst -standalone.o,.cpp,$@)

$(GAK_OBJS): CXXFLAGS += $(GAK_INCLUDES)

ifneq (,$(findstring MINGW,$(PLATFORM)))
genauthkey: $(GAK_OBJS)
	$(CXX) $(CXXFLAGS) -o $(WINBIN)/genauthkey.exe $(GAK_OBJS) $(GAK_LIBS)

install: all
else
genauthkey: libenet $(GAK_OBJS)
	$(CXX) $(CXXFLAGS) -o genauthkey $(GAK_OBJS) $(GAK_LIBS)

install: all
	cp genauthkey	../../bin_unix/$(PLATFORM_PREFIX)_genauthkey
ifneq (,$(STRIP))
	$(STRIP) ../../bin_unix/$(PLATFORM_PREFIX)_genauthkey
endif
endif

../enet/libenet.a:
	$(MAKE) -C ../enet
libenet: ../enet/libenet.a

depend:
	makedepend -a -o-standalone.o -Y -DSTANDALONE -I../shared $(subst -standalone.o,.cpp,$(GAK_OBJS))


../shared/stream-standalone.o: ../shared/cube.h ../shared/tools.h ../shared/geom.h
../shared/stream-standalone.o: ../shared/ents.h ../shared/command.h ../shared/iengine.h
../shared/stream-standalone.o: ../shared/igame.h
../shared/tools-standalone.o: ../shared/cube.h ../shared/tools.h ../shared/geom.h
../shared/tools-standalone.o: ../shared/ents.h ../shared/command.h ../shared/iengine.h
../shared/tools-standalone.o: ../shared/igame.h
../engine/genauthkey-standalone.o: ../shared/cube.h ../shared/tools.h ../shared/geom.h
../engine/genauthkey-standalone.o: ../shared/ents.h ../shared/command.h ../shared/iengine.h
../engine/genauthkey-standalone.o: ../shared/igame.h ../shared/crypto.cpp
