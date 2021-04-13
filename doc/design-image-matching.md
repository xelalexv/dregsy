# Syncing With Image Matching

## Objective
Currently, the images a task should sync need to be explicitly specified. However, often it is desirable to sync a whole set of related images with just one task, rather than having to specify each of them separately. Issues #2 and #16 are examples of this. It would be great if in a task we could simply state:

```yaml
mappings:
- from: myrepo/myproject-.*
```

Note that this is not about tags. More elaborate tag matching and filtering is going to be a different feature complementary to this one.

## Status
The design laid out in this document has been initially implemented and merged to `master`. It works for both *Docker* and *Skopeo* relays. The syntax for image matching given in the examples below may still change though. In short, consider this *alpha*.

There currently are the following known limitations and open points:

- so far only tried with *DockerHub*, *ECR*, *GCR*, and local *v2* registry
- not yet enough logging related to matching

## Approach
A pull operation itself does not support specifying any kind of selection expressions. We always have to use exact image references. So in order to sync a set of images according to some selection expression, we need to first list images of interest (which may include some coarse pre-filtering), further filter according to our needs, and finally sync all matching images.

## Matching Expressions
The `from` and `to` fields in a mapping can now contain standard [*Go* regular expressions](https://pkg.go.dev/regexp). To be recognized as such they currently need to start with prefix `regex:`.

- Regex in `from` is used for filtering items from an image list retrieved by a lister.
- Regex in `to` can be used to transform the source image path into a different target image path. It consists of two parts, the regex and a replacement expressions (see [details](https://pkg.go.dev/regexp#Regexp.ReplaceAllString)), separated by a comma. Note that this does not require `from` to be a regex, so you can use path transformations also without image matching.

    **Example:** The expression `to: regex:my(.+)/,from-dh/your$1/` would transform `myproject/webui` into `from-dh/yourproject/webui`

### Caveats
- Be careful when trying this out! Regular expressions can be surprising at times, so it would be a good idea to try them out first in a *Go* playground. You may otherwise potentially sync large numbers of images, clogging your target registry, or running into rate limits.

## Lister Types
I currently see three ways in which the initial image lists can be retrieved. Which one can be used depends on the particular registry where images are hosted, and has to be specified in the `source` section of a task.

### Lister `catalog` (default)
This uses the [`v2/_catalog`](https://docs.docker.com/registry/spec/api/#catalog) API and is mostly applicable for local registries, and for those it's often the only way in which an image list can be retrieved. It's also the default lister type and can be omitted in the `source` definition. It is important to keep in mind though that `_catalog` does not support any kind of filtering, i.e. all images are listed. It's only possible to limit the number of items to be returned in a list. For this reason, larger public registries such as *DockerHub* do not support this API. It can however be used with *AWS ECR* and *GCP GCR* registries.

Note on *ECR*: For *ECR*, pagination of list results works slightly differently than for a local registry. It requires an extra, non-standard `NextToken` parameter, which is not supported by the particular library we're using for implementing the `catalog` lister. If the registry is *ECR* we therefore automatically switch to a dedicated *ECR* lister based on the *AWS Go SDK*.

#### Examples
- This syncs all `myproject/.*` images from an *ECR* registry to a local registry. Matching images are stored with `ecr` prepended to their paths, e.g. `myproject/webui` would turn into `ecr/myproject/webui`. Note that authentication for *ECR* has to be configured as usual.

    ```yaml
    tasks:
    - name: ecr
      verbose: true
      source:
        registry: <account>.dkr.ecr.<region>.amazonaws.com
        auth-refresh: 10h
        lister:
          type: catalog # default, can be omitted
      target:
        registry: 127.0.0.1:5000
        auth: eyJ1c2VybmFtZSI6ICJhbm9ueW1vdXMiLCAicGFzc3dvcmQiOiAiYW5vbnltb3VzIn0K
        skip-tls-verify: true
      mappings:
      - from: regex:myproject/.*
        to: ecr
    ```

- This syncs all `myproject/.*` images from a *GCR* registry to a local registry. Matching images are stored with `gcr` prepended to their paths, e.g. `myproject/webui` would turn into `gcr/myproject/webui`. Note that authentication for *GCR* has to be configured as usual.

    ```yaml
    tasks:
    - name: gcr
      verbose: true
      source:
        registry: eu.gcr.io
      target:
        registry: 127.0.0.1:5000
        auth: eyJ1c2VybmFtZSI6ICJhbm9ueW1vdXMiLCAicGFzc3dvcmQiOiAiYW5vbnltb3VzIn0K
        skip-tls-verify: true
      mappings:
      - from: regex:kubika/.*
        to: gcr
    ```

### Lister `dockerhub`
As the name suggests, this lister is for getting image lists from *DockerHub*. It retrieves them via `https://hub.docker.com/v2/repositories/{user name}/`. That is, the lists that can be retrieved are limited to images of the authenticated user. Consequently, all your `from` clauses need to relate to that user. For anything else, there would be no match. Use this when you want to sync images from your own account, in particular if that includes private images.

#### Example
- This syncs all `dregsy.*` images under the *DockerHub* `xelalex` account to a local registry. Matching images are stored with `dh` prepended to their paths, e.g. `xelalex/dregsy` would turn into `dh/xelalex/dregsy`. If any private images match, they are included.

    ```yaml
    tasks:
    - name: dockerhub
      verbose: true
      source:
        registry: registry.hub.docker.com
        auth: <Dockerhub auth>
        lister:
          type: dockerhub # for including private repos
      target:
        registry: 127.0.0.1:5000
        auth: eyJ1c2VybmFtZSI6ICJhbm9ueW1vdXMiLCAicGFzc3dvcmQiOiAiYW5vbnltb3VzIn0K
        skip-tls-verify: true
      mappings:
      - from: regex:xelalex/dregsy.*
        to: dh
    ```

### Lister `index`
This uses `DefaultService.Search` from the [*Docker* registry lib](https://pkg.go.dev/github.com/docker/docker/registry). It is intended for searching larger registry sites that provide an index. For *DockerHub*, that's the `https://index.docker.io/v1/` endpoint. To perform the search, a search term needs to be given via the `search` lister property. What terms and formats can be used here again depends on the particular site. For *DockerHub* it seems to be essentially whatever you can enter into the search box on their web site. Note that it may take some time before a new image appears in a site's index, or a removed images disappears. So far, this lister has only been tested with *DockerHub*.

#### Example
- This syncs the `latest` tag of all `jenkins/jnlp-.*` images from *DockerHub* to a local registry. Paths of matching images are transformed to `dh/other-jenkins/{image}`, e.g. `jenkins/jnlp-slave` becomes `dh/other-jenkins/jnlp-slave`. Private images are not considered.

    ```yaml
    tasks:
    - name: index
      verbose: true
      source:
        registry: registry.hub.docker.com
        auth: <Dockerhub auth>
        lister:
          type: index
          search: jenkins # required for type 'index'
      target:
        registry: 127.0.0.1:5000
        auth: eyJ1c2VybmFtZSI6ICJhbm9ueW1vdXMiLCAicGFzc3dvcmQiOiAiYW5vbnltb3VzIn0K
        skip-tls-verify: true
      mappings:
      - from: regex:jenkins/jnlp-.*
        to: regex:jenkins/,dh/other-jenkins/
        tags: ["latest"]
    ```

## Note on Custom TLS Certificate Authorities
When a lister contacts an endpoint, TLS verification is based on the CA certificates offered by the host's OS. This is due to the various libraries being used to retrieve the lists. Additional CA certificates therefore need to be added using the OS's methods. Note that this is different from adding CA certificates for the *Skopeo* and *Docker* relays. There, you would place them inside `/etc/skopeo/certs.d` or `/etc/docker/certs.d`, to be used by the respective relay. They will however not be picked up by the listers.

To add CA certificates to the *dregsy* *Docker* image, you could build your own image based on it:

```Dockerfile
FROM xelalex/dregsy:latest

COPY my-ca.crt /usr/local/share/ca-certificates/my-ca.crt
RUN update-ca-certificates
```

When adding CA certificates to the host OS, you may no longer have to place them into one of the above mentioned directories for the relays, but I haven't tried that out.
