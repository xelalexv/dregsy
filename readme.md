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
    source: 
      registry: source-registry.acme.com
      auth: eyJ1c2VybmFtZSI6ICJhbGV4IiwgInBhc3N3b3JkIjogInNlY3JldCJ9Cg==
    target: 
      registry: dest-registry.acme.com
      auth: eyJ1c2VybmFtZSI6ICJhbGV4IiwgInBhc3N3b3JkIjogImFsc29zZWNyZXQifQo=
    mappings:
      - from: test/image
        to: archive/test/image
      - from: test/another-image
```

- `dockerhost` sets the *Docker* host to use as the relay
- `tasks` is a list of sync tasks, with the following settings per task: 
    - `name` for the task, required
    - `interval` in seconds at which the task should be run; when omitted, the task is only run once at start up
    - `source` and `target` describe the source and target registry for the task, with
        - `registry` pointing to the server
        - `auth` containing the credentials for the registry in the form `{"username": "...", "password": "..."}`, `base64` encoded
    - `mappings` is a list of `from`:`to` pairs that define mappings of image paths in the source registry to paths in the destination; `to` can be dropped if the path should remain the same as `from`


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
  name: kube-registry-dregsy-config
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
        - name: docker-config
          mountPath: /root/.docker
      volumes:
      - name: dregsy-config
        configMap:
          name: kube-registry-dregsy-config
          items:
          - key: config.yaml
            path: config.yaml
      - name: docker-config
        configMap:
          name: kube-registry-updater-config
          items:
          - key: docker-config.json
            path: config.json
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
