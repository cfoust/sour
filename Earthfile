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
    SAVE ARTIFACT wsproxy AS LOCAL "earthly/wsproxy"

server:
    FROM +cpp
    COPY services/server .
    RUN ./build
    SAVE ARTIFACT qserv AS LOCAL "earthly/qserv"

go:
    FROM golang:1.17
    RUN apt-get update && apt-get install -qqy libenet-dev inotify-tools
    SAVE IMAGE sour:go

relay:
    FROM +go
    COPY services/go .
    RUN ./build
    SAVE ARTIFACT relay AS LOCAL "earthly/relay"

emscripten:
    FROM emscripten/emsdk:3.1.8
    RUN apt-get update && apt-get install -y inotify-tools imagemagick
    SAVE IMAGE sour:emscripten

assets:
    ARG hash
    FROM +emscripten
    WORKDIR /tmp
    COPY services/game/assets assets
    RUN --mount=type=cache,target=/tmp/assets/working /tmp/assets/build
    SAVE ARTIFACT assets/output AS LOCAL "earthly/assets"

game:
    FROM +emscripten
    WORKDIR /cube2
    COPY services/game/cube2 /cube2
    RUN --mount=type=cache,target=/emsdk/upstream/emscripten/cache/ /cube2/build
    SAVE ARTIFACT /cube2/dist/game AS LOCAL "earthly/game"

client:
    FROM node:14.17.5
    WORKDIR /client
    COPY services/client .
    RUN --mount=type=cache,target=/code/node_modules yarn install
    RUN rm -rf dist && yarn build && cp src/index.html src/favicon.ico dist
    SAVE ARTIFACT dist AS LOCAL "earthly/client"

image-slim:
  FROM ubuntu:20.04
  # We would just use nginx:stable-alpine but the other services use some
  # dynamic libraries.
  RUN apt-get update && apt-get install -y nginx libenet-dev
  COPY +server/qserv /bin/qserv
  COPY +relay/relay /bin/relay
  COPY +proxy/wsproxy /bin/wsproxy
  COPY +game/game /app/game/
  COPY +client/dist /app/
  COPY services/ingress/production.conf /etc/nginx/conf.d/default.conf
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
  ARG tag
  SAVE IMAGE --push ghcr.io/cfoust/sour:$tag

push-slim:
  FROM +image-slim
  SAVE IMAGE --push registry.digitalocean.com/cfoust/sour:latest

push:
  FROM +image
  SAVE IMAGE --push registry.digitalocean.com/cfoust/sour:latest
