# Changelog

## `0.3.0`
- switched to *Go* 1.13 & modules
- removed *Skopeo* submodule: The *Skopeo* project is [phasing out static builds](https://github.com/containers/skopeo/issues/755), so the previous approach of building a `FROM scratch` image for *dregsy* with just the two binaries no longer works. Instead, *Alpine* is now used as the base, and *Skopeo* is installed during image build via `apk` (see `Dockerfile` for version information).

    **Important - breaking change:** The `dregsy` binary is now located at `/usr/local/bin` inside the image. You may need to adjust how you invoke *dregsy*.

- support for *AWS* China *ECR*
- doc updates
- fix for issue #4: canonicalize image refs before matching
