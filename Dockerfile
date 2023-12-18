# syntax = docker/dockerfile:1.4.0@sha256:178c4e4a93795b9365dbf6cf10da8fcf517fcb4a17f1943a775c0f548e9fc2ff

FROM debian:bookworm-slim@sha256:2bc5c236e9b262645a323e9088dfa3bb1ecb16cc75811daf40a23a824d665be9

# TODO: key goes here

RUN <<EOF cat > /etc/apt/sources.list.d/debian.sources

# Types: deb
# URIs: http://192.168.1.23:8080/debian
# Suites: bookworm bookworm-updates
# Components: main
# Signed-By: /usr/share/keyrings/debian-archive-keyring.gpg

# Types: deb
# URIs: http://192.168.1.23:8080/debian-security
# Suites: bookworm-security
# Components: main
# Signed-By: /usr/share/keyrings/debian-archive-keyring.gpg

Types: deb
URIs: http://192.168.1.23:8080/dynamic
Suites: bookworm
Components: main
Signed-By:
    -----BEGIN PGP PUBLIC KEY BLOCK-----
    
    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
    V3Ny5pXnTRp4ACE+xognKkXeXF8jn1DBc9uSCpM=
    =d/Xg
    -----END PGP PUBLIC KEY BLOCK-----

EOF

ENTRYPOINT [ "apt-get", "update" ]