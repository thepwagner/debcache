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

### TODO:

- Tracing/metrics?
- More caching of GitHub results (ideally: informed by metrics)
- More caching of cosign results (ideally: informed by metrics)
