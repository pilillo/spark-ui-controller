# Spark UI Controller

A Namespaced K8s controller exposing a route for any driver-svc binding to the Spark UI.

## Usage

Create a new role and a new SA, then bind them and start the controller as deployment:

```bash
oc apply -f role.yaml
oc apply -f service_account.yaml
oc apply -f role_binding.yaml
oc apply -f spark-ui-controller-deployment.yaml
```

Done!

## Development

### Installing the operator-sdk CLI
```bash
export ARCH=$(case $(uname -m) in x86_64) echo -n amd64 ;; aarch64) echo -n arm64 ;; *) echo -n $(uname -m) ;; esac)
export OS=$(uname | awk '{print tolower($0)}')
export OPERATOR_SDK_DL_URL=https://github.com/operator-framework/operator-sdk/releases/download/v1.13.0
curl -LO ${OPERATOR_SDK_DL_URL}/operator-sdk_${OS}_${ARCH}
chmod +x operator-sdk_${OS}_${ARCH} && sudo mv operator-sdk_${OS}_${ARCH} /usr/local/bin/operator-sdk
```

### Initiating a new controller project

```bash

Firstly, initiate a new project, under a domain and specify the repo for the go module:

export GO111MODULE=on
operator-sdk init \
--domain=github.com/pilillo \
--repo=github.com/pilillo/spark-ui-controller \
--license apache2 \
--skip-go-version-check \
--verbose
```
Notice that the last two flags can be omitted.

The project layout for Go-based operators is described [here](https://docs.okd.io/latest/operators/operator_sdk/golang/osdk-golang-project-layout.html#osdk-golang-project-layout).

Let's create a controller type:
```bash
operator-sdk create api --group=core --version=v1 --kind=Service --controller=true --resource=false
```

As opposed to creating a controller, we do not add any CRD. Therefore, there is no api folder being added.
However, if you have a look at the Dockerfile and the Makefile, most scripts expect that.
As a workaround, add an empty api folder with:

```bash
mkdir api
```

Install the openshift/api project to manage route resource types:
```bash
go get -u github.com/openshift/api
```

### Test the controller

You can run the controller on your target cluster, as defined in `~/.kube/config`:
```bash
make run
```

This clearly works with openshift either. Make sure you are in the right context and project to avoid surprises.


### Build the controller

Again, please have a look at the [Makefile](Makefile):

```bash
docker-build: test ## Build docker image with the manager.
	docker build -t ${IMG} .

docker-push: ## Push docker image with the manager.
	docker push ${IMG}
```

Therefore:

```bash
export IMG=pilillo/spark-ui-controller:v0.0.1
make docker-build
```

### Push the controller as docker image

Use the makefile:

```bash
make docker-push
```

Unless a different repo is specified in the **IMG** variable, the docker image will end up on docker hub.
For instance:

```bash
$ make docker-push
docker push pilillo/spark-ui-controller:v0.0.1
The push refers to repository [docker.io/pilillo/spark-ui-controller]
23b8cccb6fce: Pushing [=============================================>     ]  41.78MB/46.06MB
c0d270ab7e0d: Pushing [==================================================>]  3.697MB
```

Your controller is now available on Dockerhub or your private repo!

### Bundle the controller to use it with the Operator Lifecycle manager

You can use the operator sdk to build a bundle format that can be used by the operator lifecycle manager (OLM).
See the official documentation [here](https://docs.okd.io/latest/operators/operator_sdk/osdk-working-bundle-images.html).

Specifically, you can use the makefile as follows:

```bash
make bundle
```
which calls the commands `generate bundle` and `bundle validate`:

```bash
bundle: manifests kustomize ## Generate bundle manifests and metadata, then validate generated files.
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle
```

This creates:
* a bundle manifests directory at `bundle/manifests` containing a `ClusterServiceVersion` object
* a bundle metadata directory at `bundle/metadata`
* all custom resource definitions (CRDs) at `config/crd`
* a Dockerfile named `bundle.Dockerfile`

Once done, you can again use the Makefile to build and push the bundle:

```bash
.PHONY: bundle-build
bundle-build: ## Build the bundle image.
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(MAKE) docker-push IMG=$(BUNDLE_IMG)
```

Once pushed, the bundle can be deployed with:

```bash
operator-sdk run bundle \
    [-n <namespace>] \
    <registry>/<user>/<bundle_image_name>:<tag>
```

## References

* https://kubernetes.io/blog/2021/06/21/writing-a-controller-for-pod-labels/
* https://developers.redhat.com/blog/2020/02/04/how-to-use-third-party-apis-in-operator-sdk-projects#step_2__use_the_discovery_api_to_see_if_the_new_api_is_present
* https://developers.redhat.com/blog/2020/01/22/why-not-couple-an-operators-logic-to-a-specific-kubernetes-platform#how_to_implement_this_approach
* https://docs.okd.io/latest/operators/operator_sdk/golang/osdk-golang-tutorial.html
* https://docs.okd.io/latest/operators/operator_sdk/osdk-working-bundle-images.html