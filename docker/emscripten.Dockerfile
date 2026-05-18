FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y \
    build-essential \
    cmake \
    git \
    python3 \
    python3-pip \
    curl \
    ca-certificates \
    unzip \
    xz-utils \
    patch \
    pkg-config \
    imagemagick \
    inotify-tools \
    ucommon-utils \
    unrar \
    zlib1g-dev \
    libenet-dev \
    swig \
    npm \
    && rm -rf /var/lib/apt/lists/*

# Install Emscripten SDK (same version used in CI)
RUN git clone https://github.com/emscripten-core/emsdk.git /emsdk \
    && cd /emsdk \
    && ./emsdk install 3.1.8 \
    && ./emsdk activate 3.1.8

# Install modern Go toolchain (for go.mod go 1.22.x)
ENV GOVER=1.22.5
RUN curl -fsSL https://go.dev/dl/go${GOVER}.linux-amd64.tar.gz -o /tmp/go.tgz \
    && rm -rf /usr/local/go \
    && tar -C /usr/local -xzf /tmp/go.tgz \
    && rm /tmp/go.tgz

# Make Emscripten available in all shells
ENV EMSDK=/emsdk \
    EM_CONFIG=/emsdk/.emscripten \
    PATH=/emsdk:/emsdk/upstream/emscripten:/emsdk/node/14.18.2_64bit/bin:$PATH

# Prepend Go to PATH
ENV PATH=/usr/local/go/bin:$PATH

# Ensure Yarn is available in the build image (installed as root)
RUN npm i -g yarn

# Default workdir where the repo will be mounted
WORKDIR /workspace

# Show versions for easier troubleshooting
RUN emcc -v && python3 --version && cmake --version && go version

CMD ["bash"]


