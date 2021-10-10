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
    SAVE ARTIFACT wsproxy AS LOCAL "build/wsproxy"

server:
    FROM +cpp
    COPY services/server .
    RUN cmake .
    # cache cmake temp files to prevent rebuilding .o files
    # when the .cpp files don't change
    RUN --mount=type=cache,target=/code/CMakeFiles make
    SAVE ARTIFACT qserv AS LOCAL "build/qserv"

assets:
    FROM emscripten/emsdk:1.40.0
    WORKDIR /tmp
    RUN apt-get update && apt-get install -y imagemagick
    COPY services/game/assets assets
    RUN --mount=type=cache,target=/tmp/assets/working cd assets && ./package
    SAVE ARTIFACT assets/output AS LOCAL "build/assets"

game:
    FROM emscripten/emsdk:1.40.0
    WORKDIR /cube2
    COPY services/game/cube2 cube2
    RUN cd cube2/src/web && emmake make client -j8
    RUN --mount=type=cache,target=/emsdk/upstream/emscripten/cache/ mkdir -p dist/game && \
        cd cube2 && \
        # Need to get rid of some unseemly behavior
        patch sauerbraten.js file_create.patch && \
        cp -r \
          js/api.js \
          js/zee.js \
          game/zee-worker.js \
          game/gl-matrix.js \
          game/setup_low.js \
          sauerbraten.* \
          ../dist
    SAVE ARTIFACT dist AS LOCAL "build/game"

client:
    FROM node:14.17.5
    WORKDIR /client
    COPY services/client .
    RUN --mount=type=cache,target=/code/node_modules yarn install
    RUN rm -r dist && yarn build && cp src/index.html src/favicon.ico dist
    SAVE ARTIFACT dist AS LOCAL "build/client"

image-slim:
  FROM ubuntu:20.10
  # We would just use nginx:stable-alpine but the other services use some
  # dynamic libraries.
  RUN apt-get update && apt-get install -y nginx
  COPY +server/qserv /bin/qserv
  COPY +proxy/wsproxy /bin/wsproxy
  COPY +game/dist /app/game/
  COPY +client/dist /app/
  COPY services/client/nginx.conf /etc/nginx/conf.d/default.conf
  COPY services/server/config /qserv/config
  COPY entrypoint /bin/entrypoint
  CMD ["/bin/entrypoint"]
  SAVE IMAGE sour:slim

image:
  FROM +image-slim
  COPY +assets/output /app/assets/
  SAVE IMAGE sour:latest

push-slim:
  FROM +image-slim
  SAVE IMAGE --push registry.digitalocean.com/cfoust/sour:latest

push:
  FROM +image
  SAVE IMAGE --push registry.digitalocean.com/cfoust/sour:latest
