**NOTE: The container freezer functionality was removed from Knative Serving in v1.10, and this repo was archived in April 2023.**

# container-freezer

| STATUS | Sponsoring WG |
| --- | --- |
| Deprecated | [Serving](https://github.com/knative/community/blob/main/working-groups/WORKING-GROUPS.md#serving)|

A standalone service for Knative to pause/unpause containers when request count drops to zero.

**NOTE** It is highly recommended to disable aggressive probing on your application when using the container-freezer. This can be done by setting `periodSeconds: 0` on your application's readiness probe. See [here](https://github.com/psschwei/sleeptalker/blob/main/sleeptalker.yaml#L14-L15) for an example.

## Installation

### Install Knative Serving

In order to use the container-freezer, you will need a running installation of Knative Serving. Instructions for doing so can be found in the [docs](https://knative.dev/docs/admin/install/).

Note that your cluster will need to have the Service Account Token Volume Projection feature enabled. For details, see this link: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#service-account-token-volume-projection
    
Next, you will need to label your nodes with the runtime used in your cluster:

* For containerd: `kubectl label nodes minikube knative.dev/container-runtime=containerd`

* For cri-o(version>=1.24.1): `kubectl label nodes minikube knative.dev/container-runtime=crio`
    
### Install container-freezer 

```bash
RUNTIME=containerd
RELEASE=v0.1.0
kubectl apply -f "https://github.com/knative-sandbox/container-freezer/releases/download/${RELEASE}/freezer-common.yaml"
kubectl apply -f "https://github.com/knative-sandbox/container-freezer/releases/download/${RELEASE}/freezer-${RUNTIME}.yaml"
```

Note: `RUNTIME` must be one of either `containerd` or `crio`.

Check the [Releases](https://github.com/knative-sandbox/container-freezer/releases) page to get the most recent version.

### Enable concurrency endpoint in Knative Serving

By default, Knative does not enable the freezing capability. We can enable it by providing a value for `concurrency-state-endpoint` in the Knative Serving [deployment configmap](https://github.com/knative/serving/blob/main/config/core/configmaps/deployment.yaml):

``` yaml
data:
  concurrency-state-endpoint: "http://$HOST_IP:9696"
```

Alternatively, you can also patch the configmap using `kubectl`:

```bash
kubectl patch configmap/config-deployment -n knative-serving --type merge -p '{"data":{"concurrencyStateEndpoint":"http://$HOST_IP:9696"}}'
```

## Sample application

See the [sleeptalker](./test/test_images/sleeptalker/main.go) application.

