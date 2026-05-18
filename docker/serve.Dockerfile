FROM ubuntu:22.04

WORKDIR /workspace

RUN apt-get update && apt-get install -y \
    zlib1g \
    libenet7 \
    && rm -rf /var/lib/apt/lists/*

# Copy built artifacts from mounted context at build time
# Expect caller to have run build-web, build-game, build-assets, build-proxy
COPY assets/dist /workspace/assets/dist
RUN ln -s /workspace/assets/dist/.index.source /workspace/assets/.index.source

COPY pkg/server/static/site /workspace/pkg/server/static/site
COPY bin/sour /workspace/bin/sour
COPY proxy/wsproxy /workspace/proxy/wsproxy

# Entrypoint script
COPY docker/serve-entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
RUN chmod -R 777 /workspace

EXPOSE 1337

ENTRYPOINT ["/entrypoint.sh"]


