cpp:
    FROM ubuntu:20.10
    ## for apt to be noninteractive
    ENV DEBIAN_FRONTEND noninteractive
    ENV DEBCONF_NONINTERACTIVE_SEEN true
    WORKDIR /code
    RUN apt-get update && apt-get install -y build-essential cmake zlib1g-dev

proxy:
    FROM +cpp
    COPY services/proxy .
    RUN make
    SAVE ARTIFACT wsproxy AS LOCAL "wsproxy"

server:
    FROM +cpp
    COPY services/server .
    RUN cmake .
    # cache cmake temp files to prevent rebuilding .o files
    # when the .cpp files don't change
    RUN --mount=type=cache,target=/code/CMakeFiles make
    SAVE ARTIFACT qserv AS LOCAL "qserv"

client:
    FROM emscripten/emsdk:1.40.0
    WORKDIR /cube2
    COPY services/client/cube2 cube2
    RUN cd cube2/src/web && emmake make client -j8
    RUN mkdir site && \
        cp -r cube2/*.html cube2/game cube2/js cube2/*.js cube2/*.wasm cube2/*.data site && \
        mv site/bb.html site/index.html
    SAVE ARTIFACT site /site AS LOCAL "site"

docker:
  FROM ubuntu:20.10
  # For the SimpleHTTPServer
  RUN apt-get update && apt-get install -y python
  COPY +server/qserv /bin/qserv
  COPY +proxy/wsproxy /bin/wsproxy
  COPY +client/site /app/
  COPY services/server/config /qserv/config
  COPY entrypoint /bin/entrypoint
  CMD ["/bin/entrypoint"]
  SAVE IMAGE sour:latest

push:
  FROM +docker
  SAVE IMAGE --push registry.digitalocean.com/cfoust/sour:latest
