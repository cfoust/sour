TARGETS=wsproxy
CFLAGS += -fPIC

all: $(TARGETS)

wsproxy: websockify.o websocket.o base64.o md5.o sha1.o
	$(CC) $(LDFLAGS) $^ -o $@

websocket.o: websocket.c websocket.h
websockify.o: websockify.c websocket.h
md5.o: md5.c md5.h
sha1.o: sha1.c sha1.h
base64.o: base64.c

clean:
	rm -f wsproxy *.o

