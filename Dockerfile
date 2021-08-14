FROM debian@sha256:f486c516a2e06febae2e422fc251f4d29079956156a7a89410b6b21bed1ee8be

RUN apt-get update -qq && \
  apt-get install -qqy --force-yes \
    build-essential cmake && \
  rm -rf /var/lib/apt/lists/*
