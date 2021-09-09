# goflow2-loki-exporter

WIP

## Description

Push flows directly to loki. It is an alternative to sending flows to file/stdout and using promtail.

## Configuration



## Build image

(This image will contain both goflow2 and the plugin)

```bash
docker build --build-arg VERSION=`git describe --long HEAD` -t quay.io/jotak/goflow2:loki-latest .
docker push quay.io/jotak/goflow2:loki-latest

# or

podman build --build-arg VERSION=`git describe --long HEAD` -t quay.io/jotak/goflow2:loki-latest .
podman push quay.io/jotak/goflow2:loki-latest

# or with kube-enricher

podman build --build-arg VERSION=`git describe --long HEAD` -t quay.io/jotak/goflow2:kube-loki-latest -f examples/with-kube-enricher.dockerfile .
podman push quay.io/jotak/goflow2:kube-loki-latest
```

To run it, simply `pipe` goflow2 output to `loki-exporter`.

## Examples in kube

Assuming built image is `quay.io/jotak/goflow2:loki-latest`.

Since both goflow + exporter are contained inside a single image, you can declare the following command inside the pod container:

```yaml
# ...
      containers:
      - command:
        - /bin/sh
        - -c
        - /goflow2 -loglevel "trace" | /loki-exporter -loglevel "trace"
        image: quay.io/jotak/goflow2:loki-latest
# ...
```

Check the [examples](./examples) directory.

### Run on Kind with ovn-kubernetes

First, [refer to this documentation](https://github.com/ovn-org/ovn-kubernetes/blob/master/docs/kind.md) to setup ovn-k on Kind.
Then:

```bash
kubectl apply -f ./examples/goflow-kube-loki.yaml
GF_IP=`kubectl get svc goflow -ojsonpath='{.spec.clusterIP}'` && echo $GF_IP
kubectl set env daemonset/ovnkube-node -c ovnkube-node -n ovn-kubernetes OVN_IPFIX_TARGETS="$GF_IP:2055"
```

Finally check goflow's logs for output

#### Legacy Netflow (v5)

Similarly:

```bash
kubectl apply -f ./examples/goflow-kube-loki-nf5.yaml
GF_IP=`kubectl get svc goflow-leg -ojsonpath='{.spec.clusterIP}'` && echo $GF_IP
kubectl set env daemonset/ovnkube-node -c ovnkube-node -n ovn-kubernetes OVN_NETFLOW_TARGETS="$GF_IP:2056"
```


### Run on OpenShift with OVNKubernetes network provider

- Pre-requisite: make sure you have a running OpenShift cluster (4.8 at least) with `OVNKubernetes` set as the network provider.

In OpenShift, a difference with the upstream `ovn-kubernetes` is that the flows export config is managed by the `ClusterNetworkOperator`.

```bash
oc apply -f ./examples/goflow-kube-loki.yaml
GF_IP=`oc get svc goflow -ojsonpath='{.spec.clusterIP}'` && echo $GF_IP
oc patch networks.operator.openshift.io cluster --type='json' -p "$(sed -e "s/GF_IP/$GF_IP/" examples/net-cluster-patch.json)"
```

### Loki quickstart (helm)

```bash
helm upgrade --install loki grafana/loki-stack --set promtail.enabled=false
helm install loki-grafana grafana/grafana
kubectl get secret loki-grafana -o jsonpath="{.data.admin-password}" | base64 --decode ; echo
kubectl port-forward svc/loki-grafana 3000:80
```

Open http://localhost:3000/
Login with admin + printed password
Add datasource => Loki =>
http://loki:3100/

Example of queries:

- View raw logs:

`{app="goflow2"}`

- Top 10 sources by volumetry (1 min-rate):

`topk(10, (sum by(SrcWorkload,SrcNamespace) ( rate({ app="goflow2" } | json | __error__="" | unwrap Bytes [1m]) )))`

- Top 10 destinations for a given source (1 min-rate):

`topk(10, (sum by(DstWorkload,DstNamespace) ( rate({ app="goflow2",SrcNamespace="default",SrcWorkload="goflow" } | json | __error__="" | unwrap Bytes [1m]) )))`
