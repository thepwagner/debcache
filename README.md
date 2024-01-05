# debcache

Caching proxy for debian packages.

### Features:

* Acts as a pull-through cache for an existing repository.
* Acts as a dynamic repository for any set of packages:
    * Loaded from a directory on disk.
    * Attached to GitHub releases.
        * Optional `CHECKSUM.txt` verification.
        * Optional cosign verification of signed packages or signed `CHECKSUM.txt` files.


### TODO:

- Change how repo caches are configured
    * URLs are awkward
     * They are less flexible/nestable than the code affords

```yaml
repos:
  debian:
    type: cached-file
    path: ./tmp/debian
    source: meow
```

- Tracing/metrics?
- More cacheing of GitHub results (ideally: informed by metrics)