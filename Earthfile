cpp:
    FROM ubuntu:20.04
    ## for apt to be noninteractive
    ENV DEBIAN_FRONTEND noninteractive
    ENV DEBCONF_NONINTERACTIVE_SEEN true
    WORKDIR /code
    RUN apt-get update && apt-get install -y build-essential cmake zlib1g-dev inotify-tools
    SAVE IMAGE sour:cpp

proxy:
    FROM +cpp
    COPY services/proxy .
    RUN ./build
    SAVE ARTIFACT wsproxy AS LOCAL "build/wsproxy"

server:
    FROM +cpp
    COPY services/server .
    RUN ./build
    SAVE ARTIFACT qserv AS LOCAL "build/qserv"

emscripten:
    FROM emscripten/emsdk:1.39.20
    RUN apt-get update && apt-get install -y inotify-tools imagemagick
    SAVE IMAGE sour:emscripten

assets:
    ARG hash
    FROM sour:emscripten
    WORKDIR /tmp
    COPY services/game/assets assets
    RUN --mount=type=cache,target=/tmp/assets/working ./build
    SAVE ARTIFACT assets/output AS LOCAL "build/assets"

game:
    FROM sour:emscripten
    WORKDIR /cube2
    COPY services/game/cube2 cube2
    RUN --mount=type=cache,target=/emsdk/upstream/emscripten/cache/ ./build
    SAVE ARTIFACT dist AS LOCAL "build/game"

client:
    FROM node:14.17.5
    WORKDIR /client
    COPY services/client .
    RUN --mount=type=cache,target=/code/node_modules yarn install
    RUN rm -rf dist && yarn build && cp src/index.html src/favicon.ico dist
    SAVE ARTIFACT dist AS LOCAL "build/client"

image-slim:
  FROM ubuntu:20.04
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

github:
  FROM +image
  SAVE IMAGE --push ghcr.io/cfoust/sour:latest

push-slim:
  FROM +image-slim
  SAVE IMAGE --push registry.digitalocean.com/cfoust/sour:latest

push:
  FROM +image
  SAVE IMAGE --push registry.digitalocean.com/cfoust/sour:latest
