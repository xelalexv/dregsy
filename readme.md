# *dregsy* - Docker Registry Sync


## Synopsis
*dregsy* allows you to sync *Docker* images between registries, public or private. Several sync tasks can be defined, as one-off or periodic tasks (see *Configuration* section). An image is synced by using a local *Docker* daemon as a relay, i.e. the image is first pulled from the source, then tagged for the destination, and finally pushed there.


## Configuration
Sync tasks are defined in a YAML config file, e.g.:

```yaml
dockerhost: unix:///var/run/docker.sock
tasks:
  - name: task1
    interval: 60
    verbose: true
    source:
      registry: source-registry.acme.com
      auth: eyJ1c2VybmFtZSI6ICJhbGV4IiwgInBhc3N3b3JkIjogInNlY3JldCJ9Cg==
    target:
      registry: dest-registry.acme.com
      auth: eyJ1c2VybmFtZSI6ICJhbGV4IiwgInBhc3N3b3JkIjogImFsc29zZWNyZXQifQo=
    mappings:
      - from: test/image
        to: archive/test/image
        tags: ['0.1.0', '0.1.1']
      - from: test/another-image
```

- `dockerhost` sets the *Docker* host to use as the relay
- `tasks` is a list of sync tasks, with the following settings per task:
    - `name` for the task, required
    - `interval` in seconds at which the task should be run; when omitted, the task is only run once at start up
    - `verbose` determines whether for this task, more verbose output should be produced; defaults to `false` when omitted
    - `source` and `target` describe the source and target registries for the task (both required), with
        - `registry` pointing to the server, required
        - `auth` containing the credentials for the registry in the form `{"username": "...", "password": "..."}`, `base64` encoded
        - `auth-refresh` specifying an interval for automatic retrieval of credentials; only for *AWS ECR* (see below)
    - `mappings` is a list of `from`:`to` pairs that define mappings of image paths in the source registry to paths in the destination; `to` can be dropped if the path should remain the same as `from`. Additionally, the tags being synced for a mapping can be limited by providing a `tags` list.

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


## Usage

```bash
dregsy -config={path to config file}
```

If there are any periodic sync tasks defined (see *Configuration* above), *dregsy* remains running indefinitely. Otherwise, it will return once all one-off tasks have been processed.

### Running Natively
If you run *dregsy* natively on your system, the *Docker* daemon of your system will be used as the relay for all sync tasks, so all synced images will wind up in the *Docker* storage of that daemon.

### Running Inside a *Docker* Container
This will run *dregsy* inside a container (retrieved from *DockerHub*), but still use the local *Docker* daemon as the relay:

```bash
docker run --privileged --rm -v {path to config file}:/config.yaml -v /var/run/docker.sock:/var/run/docker.sock xelalex/dregsy
```

### Running On *Kubernetes (K8s)*

When you run a *Docker* registry inside your *K8s* cluster as an image cache, *dregsy* can come in handy as an automated updater for that cache. The pod in the definition below has two containers: `dind-daemon` which runs *Docker-in-Docker*, and `dregsy`, which uses `dind-daemon` as the relay.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: dregsy-config
  namespace: kube-system
data:
  config.yaml: |-
    dockerhost: tcp://localhost:2375
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
---
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
        command: ['/dregsy', '-config=/config/config.yaml']
        resources:
            requests:
                cpu: 10m
                memory: 256Mi
        volumeMounts:
        - name: dregsy-config
          mountPath: /config
      - name: dind-daemon
        image: docker:1.13.1-dind
        resources:
          requests:
            cpu: 200m
            memory: 512Mi
        securityContext:
          privileged: true
        volumeMounts:
        - name: docker-storage
          mountPath: /var/lib/docker
      volumes:
      - name: dregsy-config
        configMap:
          name: dregsy-config
          items:
          - key: config.yaml
            path: config.yaml
  volumeClaimTemplates:
    - metadata:
        name: docker-storage
      spec:
        accessModes: [ "ReadWriteOnce" ]
        resources:
          requests:
            storage: 50Gi
        selector:
          matchLabels:
            purpose: registry-updater
```


## Building

To build the *dregsy* binary, run `make build`, for building the *dregsy* *Docker* container, run `make docker`. In either case, when you build for the first time, getting vendored dependencies may take quite a while.
