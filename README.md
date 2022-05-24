# container-freezer


| STATUS | Sponsoring WG |
| --- | --- |
| Alpha | [Autoscaling](https://github.com/knative/community/blob/main/working-groups/WORKING-GROUPS.md#scaling)|

A standalone service for Knative to pause/unpause containers when request count drops to zero.

**NOTE** It is highly recommended to disable aggressive probing on your application when using the container-freezer. This can be done by setting `periodSeconds: 0` on your application's readiness probe. See [here](https://github.com/psschwei/sleeptalker/blob/main/sleeptalker.yaml#L14-L15) for an example.

## Installation

### Install Knative Serving

In order to use the container-freezer, you will need a running installation of Knative Serving. Instructions for doing so can be found in the [docs](https://knative.dev/docs/admin/install/).

Note that your cluster will need to have the Service Account Token Volume Projection feature enabled. For details, see this link: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#service-account-token-volume-projection
    
Next, you will need to label your nodes with the runtime used in your cluster:

* For containerd: `kubectl label nodes minikube knative.dev/container-runtime=containerd`

* For docker: `kubectl label nodes minikube knative.dev/container-runtime=docker`
    
### Install the container-freezer daemon

To run the container-freezer service, the first step is to clone this repository:

``` bash
git clone git@github.com:knative-sandbox/container-freezer.git
```

Next, use [`ko`](https://github.com/google/ko) to build and deploy the container-freezer service:

``` bash
cd container-freezer
ko apply -f config/
```

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

See the [sleeptalker](https://github.com/psschwei/sleeptalker) application.

