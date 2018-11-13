FROM golang:1.11

WORKDIR /go/src/vccfinder

RUN DEBIAN_FRONTEND=noninteractive apt-get update -qq && \
    DEBIAN_FRONTEND=noninteractive apt-get install -yqq \
    libssl-dev \
    libssh2-1-dev \
    libffi-dev \
    zlib1g-dev \
    build-essential \
    cmake \
    libclang-3.8-dev \
    gcc \
    pkg-config \
    git \
    libhttp-parser-dev \
    wget

RUN wget https://github.com/libgit2/libgit2/archive/v0.25.1.tar.gz && \
tar xzf v0.25.1.tar.gz && \
cd libgit2-0.25.1/ && \
cmake . && \
make && \
make install

RUN ldconfig