# Contributing to the CodeFlare Operator

Here are a few things to go over before getting started with CodeFlare Operator development:

## Environment setup

The following should be installed in your working environment:
installation/)
 - Go 1.18.X
   - [Download release](https://go.dev/dl/)
   - [Install Instructions](https://go.dev/doc/install)
 - [Operator SDK](https://sdk.operatorframework.io/docs/installation/)
 - GCC

## Basic Overview
Under the `api` dir, the MCAD and InstaScale custom resources are defined:
 - See `mcad_types.go` and `instascale_types.go`

Under `config/internal` are where the MCAD and InstaScale resource templates can be found:
 - Sorted under `mcad` and `insascale` subdirs

The code for MCAD/InstaScale resource reconsilliation can be found in the `controllers` dir:
 - See `mcad_controller.go` and `instascale_controller.go`

The main entrypoint for the operator is `main.go`

## Building and Deployment
If changes are made in the `api` dir, run: `make manifests`
 - This will generate new CRDs and associated files

If changes are made to any Go code (like in the `controllers` dir for example), run: `make`
 - This will check and build/compile the modified code

For building and pushing a new version of the operator image:
 - `make image-build -e IMG=<image-repo/image-name>`
 - `make image-push -e IMG<image-repo/image-name>`

For deploying onto a cluster:
 - First, either set `KUBECONFIG` or ensure you are logged into a cluster in your environment
 - `make install`
 - `make deploy -e IMG=<image-repo/image-name>`

## Testing
The CodeFlare Operator currently has unit tests and pre-commit checks
 - To enable and view pre-commit checks: `pre-commit install`
 - To run unit tests, run `make test-unit`

To write and inspect unit tests:
 - MCAD and InstaScale unit tests under `mcad_controller_test.go` and `instascale_controller_test.go` in the `controllers` dir
 - Unit test functions are defined in `suite_test.go` (with utils in `util/util.go`) in the `controllers dir`
 - Test cases defined under `controllers/testdata`
