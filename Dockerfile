FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    && rm -rf /var/lib/apt/lists/*

RUN curl  http://192.168.1.23:8080/debian/repo.source > /etc/apt/sources.list.d/debian.sources
RUN curl http://192.168.1.23:8080/dynamic/repo.source > /etc/apt/sources.list.d/dynamic.sources
