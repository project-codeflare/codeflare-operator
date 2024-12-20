# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=v0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=v0.0.2)
# best if we could detect this. If we cannot, we need to document it somewhere.
# then we can add a patch in the `PHONY: bundle`
# BUNDLE_VERSION is declared as bundle versioning doesn't use semver

PREVIOUS_VERSION ?= v0.0.0-dev
VERSION ?= v0.0.0-dev
BUNDLE_VERSION ?= $(VERSION:v%=%)

# APPWRAPPER_VERSION defines the default version of the AppWrapper controller
APPWRAPPER_VERSION ?= v0.27.0
APPWRAPPER_REPO ?= github.com/project-codeflare/appwrapper
APPWRAPPER_CRD ?= ${APPWRAPPER_REPO}/config/crd?ref=${APPWRAPPER_VERSION}

# KUEUE_VERSION defines the default version of Kueue (used for testing)
KUEUE_VERSION ?= v0.8.3

USE_RHOAI ?= true
# KUBERAY_VERSION defines the default version of the KubeRay operator (used for testing)
KUBERAY_VERSION ?= v1.1.0

# RAY_VERSION defines the default version of Ray (used for testing)
RAY_VERSION ?= 2.5.0

# OPERATORS_REPO_ORG points to GitHub repository organization where bundle PR is opened against
# OPERATORS_REPO_FORK_ORG points to GitHub repository fork organization where bundle build is pushed to
OPERATORS_REPO_ORG ?= redhat-openshift-ecosystem
OPERATORS_REPO_FORK_ORG ?= project-codeflare

# CHANNELS define the bundle channels used in the bundle.
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "candidate,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=candidate,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="candidate,fast,stable")
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle.
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
# To re-generate a bundle for any other default channel without changing the default setup, you can:
# - use the DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - use environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# IMAGE_ORG_BASE defines the base container registry and organization for container images.
IMAGE_ORG_BASE ?= quay.io/project-codeflare

# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
#
# For example, running 'make bundle-build bundle-push catalog-build catalog-push' will build and push both
# codeflare.dev/codeflare-operator-bundle:$VERSION and codeflare.dev/codeflare-operator-catalog:$VERSION.
IMAGE_TAG_BASE ?= $(IMAGE_ORG_BASE)/codeflare-operator

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:$(VERSION)

# BUNDLE_GEN_FLAGS are the flags passed to the operator-sdk generate bundle command
BUNDLE_GEN_FLAGS ?= -q --overwrite --version $(BUNDLE_VERSION) $(BUNDLE_METADATA_OPTS)

# USE_IMAGE_DIGESTS defines if images are resolved via tags or digests
# You can enable this value if you would like to use SHA Based Digests
# To enable set flag to true
USE_IMAGE_DIGESTS ?= false
ifeq ($(USE_IMAGE_DIGESTS), true)
	BUNDLE_GEN_FLAGS += --use-image-digests
endif

# Image URL to use all building/pushing image targets
IMG ?= ${IMAGE_TAG_BASE}:${VERSION}

# IMAGE_BUILD_FLAGS are the flags passed to the podman operator image build command
IMAGE_BUILD_FLAGS :=

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.24.2

# The target deployment environment, that corresponds to the Kustomize directory
# used to build the manifests.
ENV ?= default

# Image URL to build MNIST job test image
MNIST_JOB_TEST_VERSION ?= v0.0.2
MNIST_JOB_TEST_IMG ?= $(IMAGE_ORG_BASE)/mnist-job-test:${MNIST_JOB_TEST_VERSION}

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

BUILD_DATE := $(shell date +%Y-%m-%d\ %H:%M)
BUILD_TAG_SHA := $(shell git rev-list --abbrev-commit --tags --max-count=1)
BUILD_TAG_NAME := $(shell git describe --abbrev=0 --tags ${BUILD_TAG_SHA} 2>/dev/null || true)
BUILD_SHA := $(shell git rev-parse --short HEAD)
BUILD_VERSION := $(BUILD_TAG_NAME:v%=%)
ifneq ($(BUILD_SHA), $(BUILD_TAG_SHA))
	BUILD_VERSION := $(BUILD_VERSION)-$(BUILD_SHA)
endif
ifneq ($(shell git status --porcelain),)
	BUILD_VERSION := $(BUILD_VERSION)-dirty
endif

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

# this encounters sed issues on MacOS, quick fix is to use gsed or to escape the parentheses i.e. \( \)
.PHONY: manifests
manifests: controller-gen kustomize install-yq ## Generate RBAC objects and import upstream CRDs.
	$(CONTROLLER_GEN) rbac:roleName=manager-role webhook paths="./..."
	$(SED) -i -E "s|(- )\${APPWRAPPER_REPO}.*|\1\${APPWRAPPER_CRD}|" config/crd/appwrapper/kustomization.yaml
	$(KUSTOMIZE) build config/crd/appwrapper | $(YQ) -s '"crd-" + .spec.names.singular' --no-doc
	mv crd-*.yml config/crd

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...


##@ Build

.PHONY: modules
modules: ## Update Go dependencies.
	go get github.com/ray-project/kuberay/ray-operator@$(KUBERAY_VERSION)
	go get sigs.k8s.io/kueue@$(KUEUE_VERSION)
	go get github.com/project-codeflare/appwrapper@$(APPWRAPPER_VERSION)
	go mod tidy

.PHONY: build
build: fmt vet ## Build manager binary.
	go build \
		-ldflags " \
			-X 'main.OperatorVersion=$(BUILD_VERSION)' \
			-X 'main.BuildDate=$(BUILD_DATE)' \
		" \
		-o bin/manager main.go

.PHONY: go-build-for-image
go-build-for-image: fmt vet ## Build manager binary.
	go build \
		-ldflags " \
			-X 'main.OperatorVersion=$(BUILD_VERSION)' \
			-X 'main.BuildDate=$(BUILD_DATE)' \
		" \
		-tags strictfipsruntime -a -o manager main.go

.PHONY: run
run: manifests fmt vet ## Run a controller from your host.
	go run ./main.go

.PHONY: image-build
image-build: test-unit ## Build container image with the manager.
	podman $(IMAGE_BUILD_FLAGS) build -t ${IMG} .

.PHONY: image-push
image-push: image-build ## Push container image with the manager.
	podman push ${IMG}

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install:
	@echo "No CRDs required to be installed"
# Uncomment install and uninstall make targets once AppWrapper CRDs are required.
# .PHONY: install
# install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
# 	$(KUSTOMIZE) build config/crd | kubectl apply -f -
# 	git restore config/*

# .PHONY: uninstall
# uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
# 	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -
# 	git restore config/*

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && IMAGE=$(IMG) perl -i -pe 's/codeflare-operator-controller-image=(.*)$$/codeflare-operator-controller-image=$$ENV{"IMAGE"}/' params.env
	$(KUSTOMIZE) build config/${ENV} | kubectl apply -f -
	git restore config/*

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/${ENV} | kubectl delete --ignore-not-found=$(ignore-not-found) -f -
	git restore config/*

.PHONY: install-odh-operator
install-odh-operator: kustomize ## Install ODH operator into the OpenShift cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/odh-operator | kubectl apply -f -
	kubectl wait -n openshift-operators subscription/opendatahub-operator --for=jsonpath='{.status.state}'=AtLatestKnown --timeout=180s

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
YQ ?= $(LOCALBIN)/yq
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
GINKGO ?= $(LOCALBIN)/ginkgo
ENVTEST ?= $(LOCALBIN)/setup-envtest
OPENSHIFT-GOIMPORTS ?= $(LOCALBIN)/openshift-goimports
OPERATOR_SDK ?= $(LOCALBIN)/operator-sdk
GH_CLI ?= $(LOCALBIN)/gh
SED ?= /usr/bin/sed

## Tool Versions
KUSTOMIZE_VERSION ?= v5.4.3
CODEGEN_VERSION ?= v0.27.2
CONTROLLER_TOOLS_VERSION ?= v0.9.2
YQ_VERSION ?= v4.35.2 ## latest version that works with go1.20
OPERATOR_SDK_VERSION ?= v1.27.0
GH_CLI_VERSION ?= 2.30.0

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -s $(LOCALBIN)/kustomize || { curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

GH_CLI_DL_URL := https://github.com/cli/cli/releases/download/v$(GH_CLI_VERSION)
GH_CLI_DL_FILENAME := gh_$(GH_CLI_VERSION)_$(shell go env GOOS)_$(shell go env GOARCH)
.PHONY: install-gh-cli
install-gh-cli: $(GH_CLI)
$(GH_CLI): $(LOCALBIN)
	curl -L $(GH_CLI_DL_URL)/$(GH_CLI_DL_FILENAME).tar.gz --output $(GH_CLI_DL_FILENAME).tar.gz
	tar -xvzf $(GH_CLI_DL_FILENAME).tar.gz
	cp $(GH_CLI_DL_FILENAME)/bin/gh $(GH_CLI)
	rm -rf $(GH_CLI_DL_FILENAME)
	rm $(GH_CLI_DL_FILENAME).tar.gz

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: ginkgo
ginkgo: $(GINKGO) ## Download ginkgo locally if necessary.
$(GINKGO): $(LOCALBIN)
	test -s $(LOCALBIN)/ginkgo || GOBIN=$(LOCALBIN) go install github.com/onsi/ginkgo/v2/ginkgo

.PHONY: install-yq
install-yq: $(YQ) ## Download yq locally if necessary
$(YQ): $(LOCALBIN)
	test -s $(LOCALBIN)/yq || GOBIN=$(LOCALBIN) go install github.com/mikefarah/yq/v4@$(YQ_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@release-0.17

.PHONY: openshift-goimports
openshift-goimports: $(OPENSHIFT-GOIMPORTS) ## Download openshift-goimports locally if necessary.
$(OPENSHIFT-GOIMPORTS): $(LOCALBIN)
	test -s $(LOCALBIN)/openshift-goimports || GOBIN=$(LOCALBIN) go install github.com/openshift-eng/openshift-goimports@latest

OPERATOR_SDK_DL_URL := https://github.com/operator-framework/operator-sdk/releases/download/$(OPERATOR_SDK_VERSION)
.PHONY: install-operator-sdk
install-operator-sdk: $(OPERATOR_SDK) ## Download fixed version operator-sdk binary for consist outcome.
$(OPERATOR_SDK): $(LOCALBIN)
	curl -L $(OPERATOR_SDK_DL_URL)/operator-sdk_$(shell go env GOOS)_$(shell go env GOARCH) --output $(LOCALBIN)/operator-sdk
	chmod +x $(OPERATOR_SDK)

.PHONY: validate-bundle
validate-bundle: install-operator-sdk
	$(OPERATOR_SDK) bundle validate ./bundle --select-optional suite=operatorframework

.PHONY: bundle
bundle: manifests kustomize install-operator-sdk ## Generate bundle manifests and metadata, then validate generated files.
	cd config/manager && IMAGE=$(IMG) perl -i -pe 's/codeflare-operator-controller-image=(.*)$$/codeflare-operator-controller-image=$$ENV{"IMAGE"}/' params.env
	$(OPERATOR_SDK) generate kustomize manifests -q
	cd config/manifests && $(KUSTOMIZE) edit add patch --patch '[{"op":"add", "path":"/metadata/annotations/containerImage", "value": "$(IMG)" }]' --kind ClusterServiceVersion
	cd config/manifests && $(KUSTOMIZE) edit add patch --patch '[{"op":"add", "path":"/spec/replaces", "value": "codeflare-operator.$(PREVIOUS_VERSION)" }]' --kind ClusterServiceVersion
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle $(BUNDLE_GEN_FLAGS)
	$(MAKE) validate-bundle
	git restore config/*

.PHONY: bundle-build
bundle-build: bundle ## Build the bundle image.
	podman build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	podman push $(BUNDLE_IMG) $(BUNDLE_PUSH_OPT)

.PHONY: openshift-community-operator-release
openshift-community-operator-release: install-gh-cli bundle ## build bundle and create PR in OpenShift community operators repository
	git clone https://x-access-token:$(GH_TOKEN)@github.com/$(OPERATORS_REPO_FORK_ORG)/community-operators-prod.git
	cd community-operators-prod && git remote add upstream https://github.com/$(OPERATORS_REPO_ORG)/community-operators-prod.git && git pull upstream main && git push origin main
	cp -r bundle community-operators-prod/operators/codeflare-operator/$(BUNDLE_VERSION)
	cd community-operators-prod && git checkout -b codeflare-release-$(BUNDLE_VERSION) && git add operators/codeflare-operator/$(BUNDLE_VERSION)/* && git commit -m "add bundle manifests codeflare version $(BUNDLE_VERSION)" --signoff && git push origin codeflare-release-$(BUNDLE_VERSION)
	gh pr create --repo $(OPERATORS_REPO_ORG)/community-operators-prod --title "CodeFlare $(BUNDLE_VERSION)" --body "New release of codeflare operator" --head $(OPERATORS_REPO_FORK_ORG):codeflare-release-$(BUNDLE_VERSION) --base main
	rm -rf community-operators-prod

.PHONY: opm
OPM = ./bin/opm
opm: ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/v1.23.0/$${OS}-$${ARCH}-opm ;\
	chmod +x $(OPM) ;\
	}
else
OPM = $(shell which opm)
endif
endif

# A comma-separated list of bundle images (e.g. make catalog-build BUNDLE_IMGS=example.com/operator-bundle:v0.1.0,example.com/operator-bundle:v0.2.0).
# These images MUST exist in a registry and be pull-able.
BUNDLE_IMGS ?= $(BUNDLE_IMG)

# The image tag given to the resulting catalog image (e.g. make catalog-build CATALOG_IMG=example.com/operator-catalog:v0.2.0).
CATALOG_IMG ?= $(IMAGE_TAG_BASE)-catalog:$(VERSION)

# Set CATALOG_BASE_IMG to an existing catalog image tag to add $BUNDLE_IMGS to that image.
ifneq ($(origin CATALOG_BASE_IMG), undefined)
FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif

# Build a catalog image by adding bundle images to an empty catalog using the operator package manager tool, 'opm'.
# This recipe invokes 'opm' in 'semver' bundle add mode. For more information on add modes, see:
# https://github.com/operator-framework/community-operators/blob/7f1438c/docs/packaging-operator.md#updating-your-existing-operator
.PHONY: catalog-build
catalog-build: opm ## Build a catalog image.
	$(OPM) index add --container-tool podman --mode semver --tag $(CATALOG_IMG) --bundles $(BUNDLE_IMGS) $(FROM_INDEX_OPT)

# Build a catalog image by adding bundle images to existing catalog using the operator package manager tool, 'opm'.
.PHONY: catalog-build-from-index
catalog-build-from-index: opm ## Build a catalog image.
	mkdir catalog
	$(OPM) render $(CATALOG_BASE_IMG) -o yaml > catalog/bundles.yaml
	$(OPM) render $(BUNDLE_IMG) $(OPM_BUNDLE_OPT) > catalog/codeflare-operator-bundle.yaml
	$(SED) -i -E "s/(.*)(- name: codeflare-operator.v0.2.0)/\1- name: codeflare-operator.$(VERSION)\n  replaces: codeflare-operator.$(PREVIOUS_VERSION)\n\2/" catalog/bundles.yaml
	$(OPM) validate catalog
	$(OPM) generate dockerfile catalog
	podman build . -f catalog.Dockerfile -t $(CATALOG_IMG)

# Push the catalog image.
.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	podman push $(CATALOG_IMG) $(CATALOG_PUSH_OPT)

.PHONY: test-unit
test-unit: manifests fmt vet envtest ## Run unit tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test -v ./pkg/controllers/ -coverprofile cover.out

.PHONY: test-component
test-component: envtest ginkgo ## Run component tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" $(GINKGO) -v ./pkg/controllers/

.PHONY: test-e2e
test-e2e: manifests fmt vet ## Run e2e tests.
	go test -timeout 30m -v ./test/e2e

.PHONY: store-odh-logs
store-odh-logs: # Store all ODH relevant logs into artifact directory
	kubectl logs -n opendatahub deployment/codeflare-operator-manager > ${ARTIFACT_DIR}/codeflare-operator.log
	kubectl logs -n opendatahub deployment/kuberay-operator > ${ARTIFACT_DIR}/kuberay-operator.log
	kubectl logs -n openshift-operators deployment/opendatahub-operator-controller-manager > ${ARTIFACT_DIR}/odh-operator.log
	kubectl get events -n opendatahub > ${ARTIFACT_DIR}/odh-events.log

.PHONY: kind-e2e
kind-e2e: ## Set up e2e KinD cluster
	test/e2e/kind.sh

.PHONY: setup-e2e
setup-e2e: ## Set up e2e tests.
	KUBERAY_VERSION=$(KUBERAY_VERSION) KUEUE_VERSION=$(KUEUE_VERSION) test/e2e/setup.sh

.PHONY: imports
imports: openshift-goimports ## Organize imports in go files using openshift-goimports. Example: make imports
	$(OPENSHIFT-GOIMPORTS)

.PHONY: verify-imports
verify-imports: openshift-goimports ## Run import verifications.
	./hack/verify-imports.sh $(OPENSHIFT-GOIMPORTS)

.PHONY: image-mnist-job-test-build
image-mnist-job-test-build: ## Build container image with the MNIST job.
	podman build -t ${MNIST_JOB_TEST_IMG} ./test/pytorch_mnist_image

.PHONY: image-mnist-job-test-push
image-mnist-job-test-push: image-mnist-job-test-build ## Push container image with the MNIST job.
	podman push ${MNIST_JOB_TEST_IMG}

# Make target for generating kueue related resources
.PHONY: kueue-setup
kueue-setup:
	bash scripts/setup-kueue-resources.sh
# RHOAI/ODH related resources installation

# Basic Usage
# all-in-one will create all resources necessary to create GPU enabled ML workloads via OpenShift AI
# Users have the choice between installing RHOAI and ODH
# For RHOAI use `make all-in-one` and to remove all of the operators run `make delete-all-in-one`
# For ODH use `make all-in-one -e USE_RHOAI=false` and to remove all of the operators run `make delete-all-in-one -e USE_RHOAI=false`

##@ all-in-one
.PHONY: all-in-one
all-in-one:
	@echo -e "\n ==> Installing Everything needed for distributed AI platform on OpenShift cluster \n"
	-make install-nfd-operator
	-make install-service-mesh-operator
	-make install-ai-platform-operator
	-make install-nvidia-operator

.PHONY: delete-all-in-one
delete-all-in-one:
	@echo -e "\n ==> Removing Everything needed for distributed AI platform on OpenShift cluster \n"
	-make delete-nfd-operator
	-make delete-ai-platform-operator
	-make delete-service-mesh-operator
	-make delete-nvidia-operator

##@ general
.PHONY: delete-ai-platform-operator
delete-ai-platform-operator:
ifeq ($(USE_RHOAI), true) ## Delete RHOAI Operator
	-make delete-rhoai-operator
	-kubectl delete -f contrib/configuration/accelerator-profile.yaml -n redhat-ods-applications
else ## Delete Open Data Hub Operator
	-make delete-opendatahub-operator
	-kubectl delete -f contrib/configuration/accelerator-profile.yaml -n opendatahub
endif

.PHONY: install-ai-platform-operator
install-ai-platform-operator:
ifeq ($(USE_RHOAI), true) ## Delete RHOAI Operator
	-make install-rhoai-operator
	-kubectl apply -f contrib/configuration/accelerator-profile.yaml -n redhat-ods-applications
else ## Delete Open Data Hub Operator
	-make install-opendatahub-operator
	-kubectl apply -f contrib/configuration/accelerator-profile.yaml -n opendatahub
endif

.PHONY: delete-rhoai-operator
delete-rhoai-operator: ## Delete RHOAI Operator
	@echo -e "\n ==> Deleting OpenShift AI Operator \n"
	kubectl delete datasciencecluster/default-dsc
	kubectl wait --for=delete datasciencecluster/default-dsc --timeout=180s
	kubectl delete dsci/default-dsci
	kubectl wait --for=delete dsci/default-dsci --timeout=180s
	-kubectl delete subscription rhods-operator -n redhat-ods-operator
	-export CLUSTER_SERVICE_VERSION=`kubectl get clusterserviceversion -n redhat-ods-operator -l operators.coreos.com/rhods-operator.redhat-ods-operator -o custom-columns=:metadata.name`; \
	kubectl delete clusterserviceversion $$CLUSTER_SERVICE_VERSION -n redhat-ods-operator
	kubectl delete namespace redhat-ods-operator

.PHONY: install-rhoai-operator
install-rhoai-operator: ## Install RHOAI Operator
	@echo -e "\n ==> Installing OpenShift AI Operator \n"
	-kubectl create ns redhat-ods-operator
	kubectl create -f contrib/configuration/rhoai/rhoai-operator-subscription.yaml
	@echo Waiting for rhoai-operator Subscription to be ready
	kubectl wait -n redhat-ods-operator subscription/rhods-operator --for=jsonpath='{.status.state}'=AtLatestKnown --timeout=180s
	@while [[ -z $$(kubectl get deployment/rhods-operator -n redhat-ods-operator) ]]; do echo "."; sleep 10; done
	-export RHOAI_POD_NAME=`kubectl get -n redhat-ods-operator pod -o custom-columns=:metadata.name | grep rhods-operator`; \
	kubectl wait --for=condition=Ready pod/$$RHOAI_POD_NAME -n redhat-ods-operator
	@echo -e "\n==> Creating default Data Science Cluster \n"
	kubectl apply -f contrib/configuration/rhoai/default-dsci.yaml --server-side
	kubectl apply -f contrib/configuration/rhoai/default-dsc.yaml --server-side

.PHONY: delete-opendatahub-operator
delete-opendatahub-operator: ## Delete OpenDataHub operator
	@echo -e "\n==> Deleting OpenDataHub Operator \n"
	kubectl delete datasciencecluster/default-dsc
	kubectl wait --for=delete datasciencecluster/default-dsc --timeout=180s
	kubectl delete dsci/default-dsci
	kubectl wait --for=delete dsci/default-dsci --timeout=180s
	-kubectl delete subscription opendatahub-operator -n openshift-operators
	-export CLUSTER_SERVICE_VERSION=`kubectl get clusterserviceversion -n openshift-operators -l operators.coreos.com/opendatahub-operator.openshift-operators -o custom-columns=:metadata.name`; \
	kubectl delete clusterserviceversion $$CLUSTER_SERVICE_VERSION -n openshift-operators
	-kubectl delete namespace opendatahub

.PHONY: install-opendatahub-operator
install-opendatahub-operator: ## Install OpenDataHub operator
	@echo -e "\n==> Installing OpenDataHub Operator \n"
	-kubectl create ns opendatahub
	kubectl create -f contrib/configuration/odh/opendatahub-operator-subscription.yaml
	@echo Waiting for opendatahub-operator Subscription to be ready
	kubectl wait -n openshift-operators subscription/opendatahub-operator --for=jsonpath='{.status.state}'=AtLatestKnown --timeout=180s
	@while [[ -z $$(kubectl get deployment/opendatahub-operator-controller-manager -n openshift-operators) ]]; do echo "."; sleep 10; done
	kubectl wait --for=condition=available deployment/opendatahub-operator-controller-manager -n openshift-operators --timeout=180s
	-export ODH_POD_NAME=`kubectl get -n openshift-operators pod -o custom-columns=:metadata.name | grep opendatahub-operator-controller-manager`; \
	kubectl wait --for=condition=Ready pod/$$ODH_POD_NAME -n openshift-operators
	kubectl apply -f contrib/configuration/odh/default-dsci.yaml --server-side
	kubectl apply -f contrib/configuration/odh/default-dsc.yaml --server-side

.PHONY: delete-service-mesh-operator
delete-service-mesh-operator: ## Delete Service Mesh Operator
	@echo -e "\n==> Deleting Service Mesh Operator \n"
	kubectl delete subscription servicemeshoperator -n openshift-operators
	-export CLUSTER_SERVICE_VERSION=`kubectl get clusterserviceversion -n openshift-operators -l operators.coreos.com/servicemeshoperator.openshift-operators -o custom-columns=:metadata.name`; \
	kubectl delete clusterserviceversion $$CLUSTER_SERVICE_VERSION -n openshift-operators

.PHONY: install-service-mesh-operator
install-service-mesh-operator: ## Install Service Mesh Operator
	@echo -e "\n==> Installing OpenShift Service Mesh Operator"
	kubectl create -f contrib/configuration/service-mesh-operator-subscription.yaml
	kubectl wait -n openshift-operators subscription/servicemeshoperator --for=jsonpath='{.status.state}'=AtLatestKnown --timeout=180s
	@while [[ -z $$(kubectl get deployment/istio-operator -n openshift-operators) ]]; do echo "."; sleep 10; done
	kubectl wait --for=condition=available deployment/istio-operator -n openshift-operators --timeout=180s

##@ GPU Support
.PHONY: install-nfd-operator
install-nfd-operator: ## Install NFD operator ( Node Feature Discovery )
	@echo -e "\n==> Installing NFD Operator \n"
	-kubectl create ns openshift-nfd
	kubectl create -f contrib/configuration/nfd-operator-subscription.yaml
	@echo -e "\n==> Creating default NodeFeatureDiscovery CR \n"
	@while [[ -z $$(kubectl get customresourcedefinition nodefeaturediscoveries.nfd.openshift.io) ]]; do echo "."; sleep 10; done
	@while [[ -z $$(kubectl get csv -n openshift-nfd --selector operators.coreos.com/nfd.openshift-nfd) ]]; do echo "."; sleep 10; done
	kubectl get csv -n openshift-nfd --selector operators.coreos.com/nfd.openshift-nfd -ojsonpath={.items[0].metadata.annotations.alm-examples} | jq '.[] | select(.kind=="NodeFeatureDiscovery")' | kubectl apply -f - --validate=false

.PHONY: delete-nfd-operator
delete-nfd-operator: ## Delete NFD operator
	@echo -e "\n==> Deleting NodeFeatureDiscovery CR \n"
	kubectl delete NodeFeatureDiscovery --all -n openshift-nfd
	@while [[ -n $$(kubectl get NodeFeatureDiscovery -n openshift-nfd) ]]; do echo "."; sleep 10; done
	@echo -e "\n==> Deleting NFD Operator \n"
	-kubectl delete subscription nfd -n openshift-nfd
	-export CLUSTER_SERVICE_VERSION=`kubectl get clusterserviceversion -n openshift-nfd -l operators.coreos.com/nfd.openshift-nfd -o custom-columns=:metadata.name`; \
	kubectl delete clusterserviceversion $$CLUSTER_SERVICE_VERSION -n openshift-nfd
	-kubectl delete ns openshift-nfd

.PHONY: install-nvidia-operator
install-nvidia-operator: ## Install nvidia operator
	@echo -e "\n==> Installing nvidia Operator \n"
	-kubectl create ns nvidia-gpu-operator
	kubectl create -f contrib/configuration/nvidia-operator-subscription.yaml
	@echo -e "\n==> Creating default ClusterPolicy CR \n"
	@while [[ -z $$(kubectl get customresourcedefinition clusterpolicies.nvidia.com) ]]; do echo "."; sleep 10; done
	@while [[ -z $$(kubectl get csv -n nvidia-gpu-operator --selector operators.coreos.com/gpu-operator-certified.nvidia-gpu-operator) ]]; do echo "."; sleep 10; done
	kubectl get csv -n nvidia-gpu-operator --selector operators.coreos.com/gpu-operator-certified.nvidia-gpu-operator -ojsonpath={.items[0].metadata.annotations.alm-examples} | jq .[] | kubectl apply -f -
ifeq ($(USE_RHOAI), true) ## Additional steps required for RHOAI
	kubectl delete configmap migration-gpu-status -n redhat-ods-applications --ignore-not-found=true
	-export REPLICASET_NAME=`kubectl get replicaset -n redhat-ods-applications -l app=rhods-dashboard -o custom-columns=:metadata.name`; \
	kubectl delete replicaset $$REPLICASET_NAME -n redhat-ods-applications
endif

.PHONY: delete-nvidia-operator
delete-nvidia-operator: ## Delete nvidia operator
	@echo -e "\n==> Deleting ClusterPolicy CR \n"
	kubectl delete --ignore-not-found=true NVIDIADriver gpu-driver
	kubectl delete ClusterPolicy --all -n nvidia-gpu-operator
	@while [[ -n $$(kubectl get ClusterPolicy -n nvidia-gpu-operator) ]]; do echo "."; sleep 10; done
	@echo -e "\n==> Deleting nvidia Operator \n"
	-kubectl delete subscription gpu-operator-certified -n nvidia-gpu-operator
	-export CLUSTER_SERVICE_VERSION=`kubectl get clusterserviceversion -n nvidia-gpu-operator -l operators.coreos.com/gpu-operator-certified.nvidia-gpu-operator -o custom-columns=:metadata.name`; \
	kubectl delete clusterserviceversion $$CLUSTER_SERVICE_VERSION -n nvidia-gpu-operator
	-kubectl delete ns nvidia-gpu-operator
