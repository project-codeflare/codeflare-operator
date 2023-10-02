/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package support

import (
	"os"
)

const (
	// The environment variables hereafter can be used to change the components
	// used for testing.

	CodeFlareTestSdkVersion   = "CODEFLARE_TEST_SDK_VERSION"
	CodeFlareTestRayVersion   = "CODEFLARE_TEST_RAY_VERSION"
	CodeFlareTestRayImage     = "CODEFLARE_TEST_RAY_IMAGE"
	CodeFlareTestPyTorchImage = "CODEFLARE_TEST_PYTORCH_IMAGE"

	// The testing output directory, to write output files into.
	CodeFlareTestOutputDir = "CODEFLARE_TEST_OUTPUT_DIR"

	// The name of a secret containing InstaScale OCM token.
	InstaScaleOcmSecretName = "INSTASCALE_OCM_SECRET_NAME"
	// The namespace where a secret containing InstaScale OCM token is stored.
	InstaScaleOcmSecretNamespace = "INSTASCALE_OCM_SECRET_NAMESPACE"
)

func GetCodeFlareSDKVersion() string {
	return lookupEnvOrDefault(CodeFlareTestSdkVersion, CodeFlareSDKVersion)
}

func GetRayVersion() string {
	return lookupEnvOrDefault(CodeFlareTestRayVersion, RayVersion)
}

func GetRayImage() string {
	return lookupEnvOrDefault(CodeFlareTestRayImage, RayImage)
}

func GetPyTorchImage() string {
	return lookupEnvOrDefault(CodeFlareTestPyTorchImage, "pytorch/pytorch:1.11.0-cuda11.3-cudnn8-runtime")
}

func GetInstaScaleOcmSecretName() string {
	return lookupEnvOrDefault(InstaScaleOcmSecretName, "instascale-ocm-secret")
}

func GetInstaScaleOcmSecretNamespace() string {
	return lookupEnvOrDefault(InstaScaleOcmSecretNamespace, "default")
}

func lookupEnvOrDefault(key, value string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return value
}
