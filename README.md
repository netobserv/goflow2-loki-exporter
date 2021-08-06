# goflow2-loki-exporter

Push flows directly to loki. It is an alternative of sending flows to file/stdout and using promtail.

WIP

## Build image

(This image will contain both goflow2 and the plugin)

```bash
docker build -t quay.io/jotak/goflow:v2-loki .
docker push quay.io/jotak/goflow:v2-loki

# or

podman build -t quay.io/jotak/goflow:v2-loki .
podman push quay.io/jotak/goflow:v2-loki

# or

podman build -t quay.io/jotak/goflow:v2-kube-loki -f with-kube-enricher.dockerfile .
podman push quay.io/jotak/goflow:v2-kube-loki
```

## Run in kube

Simply `pipe` goflow2 output to `loki-exporter`.

Example of usage in kube (assuming built image `quay.io/jotak/goflow:v2-loki`)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: goflow
  name: goflow
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: goflow
  template:
    metadata:
      labels:
        app: goflow
    spec:
      containers:
      - command:
        - /bin/sh
        - -c
        - /goflow2 -loglevel "debug" | /kube-enricher | /loki-exporter
        image: quay.io/jotak/goflow:v2-kube-loki
        imagePullPolicy: IfNotPresent
        name: goflow
---
apiVersion: v1
kind: Service
metadata:
  name: goflow
  namespace: default
  labels:
    app: goflow
spec:
  ports:
  - port: 2055
    protocol: UDP
  selector:
    app: goflow
```
