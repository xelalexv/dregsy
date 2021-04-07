# *dregsy* - Docker Registry Sync

## Synopsis
*dregsy* lets you sync *Docker* images between registries, public or private. Several sync tasks can be defined, as one-off or periodic tasks (see *Configuration* section). An image is synced by using a *sync relay*. Currently, this can be either [*Skopeo*](https://github.com/containers/skopeo) or a local *Docker* daemon. When using the latter, the image is first pulled from the source, then tagged for the destination, and finally pushed there. *Skopeo* in contrast, can directly transfer an image from source to destination, which makes it the preferred choice.


## Configuration
Sync tasks are defined in a YAML config file:

```yaml
# relay type, either 'skopeo' or 'docker'
relay: skopeo

# relay config sections
skopeo:
  # path to the skopeo binary; defaults to 'skopeo', in which case it needs to
  # be in PATH
  binary: skopeo
  # directory under which to look for client certs & keys, as well as CA certs
  # (see note below)
  certs-dir: /etc/skopeo/certs.d

docker:
  # Docker host to use as the relay
  dockerhost: unix:///var/run/docker.sock
  # Docker API version to use, defaults to 1.24
  api-version: 1.24

# settings for image matching (see below)
lister:
  # maximum number of repositories to list, set to -1 for no limit, defaults to 100
  maxItems: 100
  # for how long a repository list will be re-used before retrieving again;
  # specify as a Go duration value ('s', 'm', or 'h'), set to -1 for not caching,
  # defaults to 1h
  cacheDuration: 1h

# list of sync tasks
tasks:

  - name: task1 # required

    # interval in seconds at which the task should be run; when omitted,
    # the task is only run once at start-up
    interval: 60

    # determines whether for this task, more verbose output should be
    # produced; defaults to false when omitted
    verbose: true

    # 'source' and 'target' are both required and describe the source and
    # target registries for this task:
    #  - 'registry' points to the server; required
    #  - 'auth' contains the base64 encoded credentials for the registry
    #    in JSON form {"username": "...", "password": "..."}
    #  - 'auth-refresh' specifies an interval for automatic retrieval of
    #    credentials; only for AWS ECR (see below)
    #  - 'skip-tls-verify' determines whether to skip TLS verification for the
    #    registry server (only for 'skopeo', see note below); defaults to false
    source:
      registry: source-registry.acme.com
      auth: eyJ1c2VybmFtZSI6ICJhbGV4IiwgInBhc3N3b3JkIjogInNlY3JldCJ9Cg==
    target:
      registry: dest-registry.acme.com
      auth: eyJ1c2VybmFtZSI6ICJhbGV4IiwgInBhc3N3b3JkIjogImFsc29zZWNyZXQifQo=
      skip-tls-verify: true

    # 'mappings' is a list of 'from':'to' pairs that define mappings of image
    # paths in the source registry to paths in the destination; 'from' is
    # required, while 'to' can be dropped if the path should remain the same as
    # 'from'. Regular expressions are supported in both fields (read on below
    # for more details). Additionally, the tags being synced for a mapping can
    # be limited by providing a 'tags' list. When omitted, all image tags are
    # synced.
    mappings:
      - from: test/image
        to: archive/test/image
        tags: ['0.1.0', '0.1.1']
      - from: test/another-image
```


### Caveats

When syncing via a *Docker* relay, do not use the same *Docker* daemon for building local images (even better: don't use it for anything else but syncing). There is a risk that the reference to a locally built image clashes with the shorthand notation for a reference to an image on `docker.io`. E.g. if you built a local image `busybox`, then this would be indistinguishable from the shorthand `busybox` pointing to `docker.io/library/busybox`. One way to avoid this is to use `registry.hub.docker.com` instead of `docker.io` in references, which would never get shortened. If you're not syncing from/to `docker.io`, then all of this is not a concern.

### Image Matching

The `mappings` section of a task can employ *Go* regular expressions for describing what images to sync, and how to change the destination path and name of an image. Details about how this works and examples can be found in this [design document](doc/design-image-matching.md). Note however that this is still an *alpha* feature, so things may not quite work as expected. Also keep in mind that regular expressions can be surprising at times, so it would be a good idea to try them out first in a *Go* playground. You may otherwise potentially sync large numbers of images, clogging your target registry, or running into rate limits. Feedback about this feature is encouraged! 

### Repository Validation & Client Authentication with TLS

When connecting to source and target repository servers, TLS validation is performed to verify the identity of a server. If you're using self-signed certificates for a repo server, or a server's certificate cannot be validated with the CA bundle available on your system, you need to provide the required CA certs. The *dregsy* *Docker* image includes the CA bundle that comes with the *Alpine* base image. Also, if a repo server requires client authentication, i.e. mutual TLS, you need to provide an appropriate client key & cert pair.

How you do that for *Docker* is [described here](https://docs.docker.com/engine/security/certificates/). The short version: create a folder under `/etc/docker/certs.d` with the same name as the repo server's host name, e.g. `source-registry.acme.com`, and place any required CA certs there as `*.crt` (mind the extension). Client key & cert pairs go there as well, as `*.key` and `*.cert`.

Example:

```
/etc/docker/certs.d/
    └── source-registry.acme.com
       ├── client.cert
       ├── client.key
       └── ca.crt
```

When using the `skopeo` relay, this is essentially the same, except that you specify the root folder with the `skopeo` setting `certs-dir` (defaults to `/etc/skopeo/certs.d`). However, it's important to note the following differences:

- When a repo server uses a non-standard port, the port number is included in image references when pulling and pushing. For TLS validation, `docker` will accordingly expect a `{registry host name}:{port}` folder. For `skopeo`, this is not the case, i.e. the port number is dropped from the folder name. This was a conscious decision to avoid pain when running *dregsy* in *Kubernetes* and mounting certs & keys from secrets: [mount paths must not contain `:`](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#volumemount-v1-core).

- To skip TLS verification for a particular repo server when using the `docker` relay, you need to [configure the *Docker* daemon accordingly](https://docs.docker.com/registry/insecure/). With `skopeo`, you can easily set this in any source or target definition with the `skip-tls-verify` setting.


### *AWS ECR*

If a source or target is an *AWS ECR* registry, you need to retrieve the `auth` credentials via *AWS CLI*. They would however only be good for 12 hours, which is ok for one off tasks. For periodic tasks, or to avoid retrieving the credentials manually, you can specify an `auth-refresh` interval as a *Go* `Duration`, e.g. `10h`. If set, *dregsy* will initially and whenever the refresh interval has expired retrieve new access credentials. `auth` can be omitted when `auth-refresh` is set. Setting `auth-refresh` for anything other than an *AWS ECR* registry will raise an error.

Note however that you either need to set environment variables `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` for the *AWS* account you want to use and a user with sufficient permissions. Or if you're running *dregsy* on an *EC2* instance in your *AWS* account, the machine should have an appropriate instance profile. An according policy could look like this:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ecr:GetAuthorizationToken",
        "ecr:CreateRepository"
      ],
      "Resource": ["*"]
    },
    {
      "Effect": "Allow",
      "Action": [
        "ecr:GetDownloadUrlForLayer",
        "ecr:BatchGetImage",
        "ecr:BatchCheckLayerAvailability",
        "ecr:DescribeRepositories",
        "ecr:PutImage",
        "ecr:InitiateLayerUpload",
        "ecr:UploadLayerPart",
        "ecr:CompleteLayerUpload"
      ],
      "Resource": "arn:aws:ecr:<your_region>:<your_account>:repository/*"
    }
  ]
}
```

### *GCR (Google Cloud Platform)*

If a source or target is a *Google Container Registry (GCR)*, `auth` may be omitted altogether. In this case either `GOOGLE_APPLICATION_CREDENTIALS` variable must be set (which is supposed to contain a path to a JSON file with credentials for a *GCP* service account), or *dregsy* must be run on a *GCE* instance with an appropriate service account attached. In case of *GCR*, `registry` must be specified as any of *GCR* addresses (i.e. `gcr.io`, `us.gcr.io`, `eu.gcr.io`, or `asia.gcr.io`), while the `from/to` mapping must include your *GCP* project name (i.e. `your-project-123/your-image`). Note that `GOOGLE_APPLICATION_CREDENTIALS`, if set, takes precedence even on a *GCE* instance.

If you want to use *GCR* as the source for a public image, you can deactivate authentication all together by setting `auth` to `none`.

## Usage

```bash
dregsy -config={path to config file}
```

If there are any periodic sync tasks defined (see *Configuration* above), *dregsy* remains running indefinitely. Otherwise, it will return once all one-off tasks have been processed.

### Logging
Logging behavior can be changed with these environment variables:

| variable     | function   | values                                            |
|--------------|------------|---------------------------------------------------|
| `LOG_LEVEL`  | log level; defaults to `info` | `fatal`, `error`, `warn`, `info`, `debug`, `trace`|
| `LOG_FORMAT` | log format; gets automatically switched to *JSON* when *dregsy* is run without a TTY | `json` to force *JSON* log format, `text` to force text output |
| `LOG_FORCE_COLORS` | force colored log messages when running with a TTY | `true`, `false` |
| `LOG_METHODS` | include method names in log messages | `true`, `false` |

### Running Natively
If you run *dregsy* natively on your system, with relay type `docker`, the *Docker* daemon of your system will be used as the relay for all sync tasks, so all synced images will wind up in the *Docker* storage of that daemon.

### Running Inside a *Docker* Container
You can use the [*dregsy* image on Dockerhub](https://hub.docker.com/r/xelalex/dregsy/) for running *dregsy* containerized. There are two variants: one is based on *Alpine*, and suitable when you just want to run *dregsy*. The other variant is based on *Ubuntu*. It's somewhat larger, but may be better suited as a base when you want to extend the *dregsy* image. It's often easier to add things there than on *Alpine*, e.g. the *AWS* command line interface.

With each release, three tags get published: `{version}-ubuntu`, `{version}-alpine`, and `{version}`, with the latter two referring to the same image. The same applies for `latest`. The *Skopeo* versions contained in the two variants may not always be exactly the same, but should only differ in patch level.

#### With `skopeo` relay
The image includes the `skopeo` binary, so all that's needed is:

```bash
docker run --rm -v {path to config file}:/config.yaml xelalex/dregsy
```

#### With `docker` relay
This will still use the local *Docker* daemon as the relay:

```bash
docker run --privileged --rm -v {path to config file}:/config.yaml -v /var/run/docker.sock:/var/run/docker.sock xelalex/dregsy
```

### Running On *Kubernetes*

When you run a *Docker* registry inside your *Kubernetes* cluster as an image cache, *dregsy* can come in handy as an automated updater for that cache. The example config below uses the `skopeo` relay:

```yaml
relay: skopeo
tasks:
  - name: task1
    interval: 60
    source:
      registry: registry.acme.com
      auth: eyJ1c2VybmFtZSI6ICJhbGV4IiwgInBhc3N3b3JkIjogInNlY3JldCJ9Cg==
    target:
      registry: registry.my-cluster
      auth: eyJ1c2VybmFtZSI6ICJhbGV4IiwgInBhc3N3b3JkIjogImFsc29zZWNyZXQifQo=
    mappings:
      - from: test/image
        to: archive/test/image
      - from: test/another-image
```

To keep your registry auth tokens in the config file secure, we are creating a Kubernetes _Secret_ instead of a _ConfigMap_:

```sh
kubectl create secret generic dregsy-config --from-file=./config.yaml
```

In addition, you will most likely want to mount client certs & keys, and CA certs from *Kubernetes* secrets into the pod for TLS validation to work. (The CA bundle from the official `golang` image is already included in the *dregsy* image.)

```yaml
apiVersion: apps/v1beta1
kind: StatefulSet
metadata:
  name: kube-registry-updater
  namespace: kube-system
  labels:
    k8s-app: kube-registry-updater
    kubernetes.io/cluster-service: "true"
spec:
  serviceName: kube-registry-updater
  replicas: 1
  template:
    metadata:
      labels:
        k8s-app: kube-registry-updater
        kubernetes.io/cluster-service: "true"
    spec:
      containers:
      - name: dregsy
        image: xelalex/dregsy
        command: ['dregsy', '-config=/config/config.yaml']
        resources:
          requests:
            cpu: 10m
            memory: 32Mi
        volumeMounts:
        - name: dregsy-config
          mountPath: /config
          readOnly: true
      volumes:
      - name: dregsy-config
        secret:
          secretName: dregsy-config
```


## Development

### Building

The `Makefile` has targets for building the binary and *Docker* image, and other stuff. Just run `make` to get a list of the targets, and info about configuration items. Note that for consistency, building is done inside a *Golang* build container, so you will need *Docker* to build. *dregsy*'s *Docker* image is based on *Alpine*, and installs *Skopeo* via `apk` during the image build.

### Testing

Tests are also started via the `Makefile`. To run the tests, you will need to prepare the following:

- Configure the *Docker* daemon: The tests run containerized, but need access to the local *Docker* daemon for testing the *Docker* relay. One way is to mount the `/var/run/docker.socks` socket into the container (the `Makefile` takes care of that). However, the `docker` group on the host may not map onto the group of the user inside the testing container. The preferred way is therefore to let the *Docker* daemon listen on `127.0.0.1:2375`. Since the testing container runs with host network, the tests can access this directly. Decide which setup to use and configure the *Docker* daemon accordingly. Additionally, set it to accept `127.0.0.1:5000` as an insecure registry.

- An *AWS* account to test syncing with *ECR*: Create a technical user in that account. This user should have full *ECR* permissions, i.e. the `AmazonEC2ContainerRegistryFullAccess` policy attached, since it will delete the used repo after the tests are done.

- A *Google Cloud* account to test syncing with *GCR*: Create a project with the *Container Registry* API enabled. In that project, you need a service account with the roles *Cloud Build Service Agent* and *Storage Object Admin* enabled, since this service account also will need to delete the synced images again after the tests.

The details for above requirements are configured via a `.makerc` file in the root of this project. Just run `make` and check the *Notes* section in the help output. Here's an example:

```make
# Docker config; to use the Unix socket, set to unix:///var/run/docker.sock
DREGSY_TEST_DOCKERHOST = tcp://127.0.0.1:2375

# ECR
DREGSY_TEST_ECR_REGISTRY = {account ID}.dkr.ecr.eu-central-1.amazonaws.com
DREGSY_TEST_ECR_REPO = dregsy/test
AWS_ACCESS_KEY_ID = {key ID}
AWS_SECRET_ACCESS_KEY = {access key}

# GCR
DREGSY_TEST_GCR_HOST = eu.gcr.io
DREGSY_TEST_GCR_PROJECT = {your project}
DREGSY_TEST_GCR_IMAGE = dregsy/test
GCP_CREDENTIALS = {full path to access JSON of service account}
```
