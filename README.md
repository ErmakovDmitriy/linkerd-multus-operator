# linkerd-multus-operator

This operator manager Multus NetworkAttachmentDefinition resources based on Linkerd AttachDefinition
resources in a Namespace.

The intention is to provide support for Linkerd in Openshift environment.
Openshift uses Multus which does not support CNI chaining, so the CNIs must be called as NetworkAttachmentDefinitions.

## Description

The project defines a CRD [AttachDefinition](api/v1alpha1/attachdefinition_types.go)
which at the moment just shows to the operator that the NetworkAttachmentDefinition must be created.

The operator also provides a mutating webhook to annotate Pods which have `linkerd.io/inject`
annotation with `k8s.v1.cni.cncf.io/networks` annotation to make Multus call linkerd-cni plugin.

To use this operator a customized linkerd-cni must be deployed in an Openshift cluster from
`https://github.com/ErmakovDmitriy/linkerd2` repository. A build is provided at
[Docker Hub](https://hub.docker.com/repository/docker/demonihin/linkerd2-cni).

The only change from the upstream Linkerd CNI is that the customized plugin returns a dummy CNI JSON result,
if nothing is provided from a previous plugin (support to be called as a stand-alone, not chained).

The built container image is available at [Docker Hub](https://hub.docker.com/repository/docker/demonihin/linkerd-multus-operator).

This application should be treated just as a proof of concept.

A future idea is to modify linkerd proxy-injector to support a Pod proxy customizations
based on the AttachDefinition resource which may improve Openshift support because the default
settings (specifically a static ProxyUID) are not allowed by default in Openshift.

## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Running on the cluster
1. Install Instances of Custom Resources:

```sh
kubectl apply -f config/samples/
```

2. Build and push your image to the location specified by `IMG`:
	
```sh
make docker-build docker-push IMG=<some-registry>/linkerd-multus-operator:tag
```
	
3. Deploy the controller to the cluster with the image specified by `IMG`:

```sh
make deploy IMG=<some-registry>/linkerd-multus-operator:tag
```

### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller
UnDeploy the controller to the cluster:

```sh
make undeploy
```

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/) 
which provides a reconcile function responsible for synchronizing resources untile the desired state is reached on the cluster 

### Test It Out
1. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

