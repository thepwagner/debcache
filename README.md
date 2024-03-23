# debcache

Caching proxy for debian packages.

### Features:

* Acts as a pull-through cache for existing repositories.
* Acts as a dynamic repository for any set of packages:
    * Lists debs in a directory on disk.
    * Discovers debs attached to releases as a GitHub repository.
        * Optional `CHECKSUM.txt` verification.
        * Optional cosign verification of signed packages or signed `CHECKSUM.txt` files.
        * Clearly optimized for `goreleaser` projects ❤️.

### Testing

The default configuration expects a GPG private key at `tmp/key.asc`. You can generate one like this:

```bash
mkdir -p tmp/gpg
GNUPGHOME=tmp/gpg gpg --batch --gen-key <<EOF
%no-protection
Key-Type: RSA
Key-Length: 2048
Name-Real: Debcache
Name-Email: fake@debcache.dev
Expire-Date: 0
%commit
EOF

GNUPGHOME=tmp/gpg gpg --export-secret-key --armor > tmp/key.asc
rm -Rf tmp/gpg
```

The server can generate `.sources` configuration, so configuring a container to use the `github` repository in the test server looks like:

```dockerfile
FROM debian:bookworm-slim
RUN apt-get update && \
    apt-get install -y  --no-install-recommends \
      curl \
    && rm -rf /var/lib/apt/lists/* /tmp/*

RUN curl http://MY_HOST_IP:8080/github/repo.source > /etc/apt/sources.list.d/github.sources
```
