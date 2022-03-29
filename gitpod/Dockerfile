FROM gitpod/workspace-full

RUN sudo apt-get update && \
  sudo apt-get install -o Dpkg::Options::="--force-confold" -qqy \
    build-essential \
    cmake \
    imagemagick \
    inotify-tools \
    libenet-dev \
    nginx \
    zlib1g-dev \
    && \
  sudo rm -rf /var/lib/apt/lists/*

RUN cd /home/gitpod && \
  git clone https://github.com/emscripten-core/emsdk.git && \
  cd emsdk && \
  git checkout 0ea8f8a8707070e9a7c83fbb4a3065683bcf1799 && \
  ./emsdk install 1.39.20 && \
  ./emsdk activate 1.39.20 && \
  bash -c 'source /home/gitpod/emsdk/emsdk_env.sh && npm install -g yarn@1.22.5' && \
  echo 'source "/home/gitpod/emsdk/emsdk_env.sh"' >> /home/gitpod/.bashrc

RUN /bin/bash -c 'source /home/gitpod/.nvm/nvm.sh && nvm install 14.17.5 && nvm alias default 14.17.5' && \
  echo 'source "/home/gitpod/.nvm/nvm.sh"' >> /home/gitpod/.bashrc
