# syntax = docker/dockerfile:1.4.0

FROM debian:bookworm-slim

RUN <<EOF cat > /etc/apt/sources.list.d/debian.sources
Types: deb
URIs: http://192.168.1.23:8080/debian
Suites: bookworm
Components: main
Signed-By: /usr/share/keyrings/debian-archive-keyring.gpg
EOF

ENTRYPOINT [ "apt-get", "update" ]