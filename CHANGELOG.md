# Changelog

## post-`0.4.5` after merging support for pulling with image digests wit skopeo
- For Skopeo relay, adding support for pulling container images based on the image digest. An image digest looks like this: `sha256:5e8e0509e829bb8f990249135a36e81a3ecbe94294e7a185cc14616e5fad96bd`.
  Below is the behavior of the `skopeo copy` Sync with support for both
  digests and tags.
  Warning: Skopeo Docker references with both a tag and a digest are
  currently not supported.

  The `digests` list & `tags` list are stored in a `mapping` struct.
  As example, look at the file
  `test/fixtures/config/skopeo-digest-only-valid.yaml`

  | `digests` list | `tags` list | `dregsy` behavior                             | diff with 0.4.4 |
  |----------------|-------------|-----------------------------------------------|-----------------|
  | empty          | empty       | pulls all tags                                | same            |
  | empty          | NOT empty   | pulls filtered tags only                      | same            |
  | NOT empty      | NOT empty   | pulls filtered tags AND pulls correct digests | different       |
  | NOT empty      | empty       | pulls correct digests only, ignores tags      | different       |

  A "correct digest" is a correctly formated AND an existing digest.
  Skopeo is used to verify if the digest exists before trying to copy it.
- Adding a simple bash script that uses `openssl` and `podman` to instantiate a simple registryv2 service with an HTTPS endpoint: `test/bin/create_local_registry_https.sh`, for testing purpose.
- Adding `dregsy` configuration samples, for testing purpose:
    + `skopeo-digest-bad-formated.yaml`: a config sample with image digests that are badly formated. Dregsy should raise an error in logs and skip to next tag or digest.
    + `skopeo-digest-dont-exist.yaml`: a config sample with image digests that are formated correctly, but that does not exist. Dregsy should raise an error in logs and skip to next tag or digest.
    + `skopeo-digest-duplicates.yaml`: a config sample with duplicates image digests. Dregsy should eliminate duplicates and only copy once.
    + `skopeo-digest-only-valid.yaml`: a config sample with valid image digests that exist on docker hub (`docker.io`). Dregsy should copy them without issue.
- Adding 2 `podman` or `docker` Containerfiles for x86 and ARM64 architecture: `Containerfile.aarch64-arm64` and `Containerfile.x86_64-amd64`. For testing purpose and local build.
- Adding a section in the README: _### Building only the `dregsy` binary with `podman`_
- upgrades:
    + *Go* 1.20.2 in `Containerfile.aarch64-arm64`, `Containerfile.x86_64-amd64`.


## `0.4.5`
- upgrades:
    + *Go* 1.20.1
    + latest *Ubuntu 22.04* and *Alpine 3.17*
    + *Skopeo* 1.11.1
    + misc. lib upgrades for CVE remediation

## `0.4.4`
- use `Metadata-Flavor` header when checking for *Google* metadata server (pr #82)
- upgrades:
    + *Go* 1.18.8
    + latest *Ubuntu 22.04* (remediates [CVE-2022-3602](https://ubuntu.com/security/CVE-2022-3602) and [CVE-2022-3786](https://ubuntu.com/security/CVE-2022-3786))
    + *Alpine 3.16*
    + *Skopeo* 1.9.3

## `0.4.3`
- support for pruning filtered tag sets (issue #72, *alpha* feature)
- added `-run` option for filtering tasks to run (issue #59)
- support for platform selection when syncing from multi-platform images (issue #43, *alpha* feature)
- raised default *Docker* API version to `1.41`
- adjusted build to also work on MacOS+M1
- remediation of CVEs in dependencies:
    + [Improper Input Validation in GoGo Protobuf](https://github.com/advisories/GHSA-c3h9-896r-86jm)
    + [containerd CRI plugin: Insecure handling of image volumes](https://github.com/advisories/GHSA-crp2-qrr5-8pq7)
    + [containerd CRI plugin: Host memory exhaustion through ExecSync](https://github.com/advisories/GHSA-5ffw-gxpp-mxpf)
- upgrades:
    + *Go* 1.18
    + *Ubuntu 22.04* and *Alpine 3.15*
    + *Skopeo* 1.8.0 (own build)

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
