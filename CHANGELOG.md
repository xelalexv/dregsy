# Changelog

## `master`
- remediation of CVEs in dependencies:
    + [Improper Input Validation in GoGo Protobuf](https://github.com/advisories/GHSA-c3h9-896r-86jm)
    + [containerd CRI plugin: Insecure handling of image volumes](https://github.com/advisories/GHSA-crp2-qrr5-8pq7)
- upgrades:
    + switched to *Go* 1.18
    + switched to *Ubuntu 22.04* and *Alpine 3.15*
    + *Skopeo* to 1.5.2 (*Alpine*) & 1.4.2 (*Ubuntu*)

        *Note*: Due to switching to 22.04, we can now install *Skopeo* on *Ubuntu* from OS packages. Latest version there is 1.4.2. *Alpine* however skipped from 1.3 to 1.5, so for the time being, we will have to live with different minor versions of *Skopeo* in the two images.

## `0.4.2`
- remediation of CVEs in dependencies:
    + [OCI Manifest Type Confusion Issue](https://github.com/advisories/GHSA-qq97-vm5h-rrhg)
    + [Ambiguous OCI manifest parsing](https://github.com/advisories/GHSA-5j5w-g665-5m35)
    + [Clarify `mediaType` handling](https://github.com/advisories/GHSA-77vh-xpmg-72qh)
    + [Insufficiently restricted permissions on plugin directories](https://github.com/advisories/GHSA-c2h3-6mxw-7mvq)
- upgrades:
    + switched to *Go* 1.17
    + *Ubuntu 20.04* and *Alpine 3.14* to latest container images
- fixes:
    + building on non-*Linux* platforms (issue #61)

## `0.4.1`
- remediation of CVEs in dependencies:
    + [CVE-2020-26160](https://github.com/advisories/GHSA-w73w-5m7g-f7qc), `github.com/dgrijalva/jwt-go`
    + [GHSA-c72p-9xmj-rx3w](https://github.com/advisories/GHSA-c72p-9xmj-rx3w), `github.com/containerd/containerd`
- upgrades:
    + *Skopeo* to 1.3.1 (*Alpine*) & 1.3.0 (*Ubuntu*)
    + *Alpine* to 3.14.0
    + *Ubuntu 20.04* to latest container image

## `0.4.0`
- support for image matching (issue #16, *alpha* feature)
- tag filtering with *semver* and *regex* (issue #22, *alpha* feature)
- support token based authentication for *Google Artifact Registry* (issue #51)
- doc updates & corrections

## `0.3.6`
- added container image based on *Ubuntu 20.04* (issue #47)

## `0.3.5`
- upgraded to *Alpine* 3.13.1 & *Skopeo* 1.2.1 (issue #29)

## `0.3.4`
- allow to deactivate authentication for public images on *GCR* (issue #37)

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
