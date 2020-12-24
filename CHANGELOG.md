# Changelog

## `0.3.3`
- fixed stopping of one-off tasks (issue #35)

## `0.3.2`
- support for *Google Container Registry* (issue #30)
- switched to `logrus` for logging (issue #32)
- added basic e2e tests (issue #28)
- code refactored

## `0.3.1`
- added more info to error messages during image ref matching (*Docker* relay, issue #18)
- upgraded to *Skopeo* 0.2.0, switched to using *Skopeo*'s `list-tags` command (issue #13)
- documentation updates

## `0.3.0`
- switched to *Go* 1.13 & modules
- removed *Skopeo* submodule: The *Skopeo* project is [phasing out static builds](https://github.com/containers/skopeo/issues/755), so the previous approach of building a `FROM scratch` image for *dregsy* with just the two binaries no longer works. Instead, *Alpine* is now used as the base, and *Skopeo* is installed during image build via `apk` (see `Dockerfile` for version information).

    **Important - breaking change:** The `dregsy` binary is now located at `/usr/local/bin` inside the image. You may need to adjust how you invoke *dregsy*.

- support for *AWS* China *ECR*
- doc updates
- fix for issue #4: canonicalize image refs before matching
