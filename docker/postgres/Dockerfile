FROM postgres:15

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential gcc make git postgresql-server-dev-15 \
    && rm -rf /var/lib/apt/lists/*

RUN git clone https://github.com/amutu/zhparser.git \
    && cd zhparser \
    && make && make install \
    && rm -rf /zhparser
