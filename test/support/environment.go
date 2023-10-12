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
	"strings"
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

	// The namespace where a secret containing InstaScale OCM token is stored and the secret name.
	InstaScaleOcmSecret = "INSTASCALE_OCM_SECRET"

	// Cluster ID for OSD cluster used in tests, used for testing InstaScale
	OsdClusterID = "CLUSTERID"

	// Type of cluster test is run on
	ClusterTypeEnvVar = "CLUSTER_TYPE"
)

type ClusterType string

const (
	OsdCluster        ClusterType = "OSD"
	OcpCluster        ClusterType = "OCP"
	HypershiftCluster ClusterType = "HYPERSHIFT"
	UndefinedCluster  ClusterType = "UNDEFINED"
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

func GetInstascaleOcmSecret() (string, string) {
	res := strings.SplitN(lookupEnvOrDefault(InstaScaleOcmSecret, "default/instascale-ocm-secret"), "/", 2)
	return res[0], res[1]
}

func GetOsdClusterId() (string, bool) {
	return os.LookupEnv(OsdClusterID)
}

func GetClusterType(t Test) ClusterType {
	clusterType, ok := os.LookupEnv(ClusterTypeEnvVar)
	if !ok {
		t.T().Logf("Expected environment variable %s not found, cluster type is not defined.", ClusterTypeEnvVar)
		return UndefinedCluster
	}
	switch clusterType {
	case "OSD":
		return OsdCluster
	case "OCP":
		return OcpCluster
	case "HYPERSHIFT":
		return HypershiftCluster
	default:
		t.T().Logf("Expected environment variable %s contains unexpected value: '%s'", ClusterTypeEnvVar, clusterType)
		return UndefinedCluster
	}
}

func IsOsd() bool {
	osdClusterId, found := GetOsdClusterId()
	if found && osdClusterId != "" {
		return true
	}
	return false
}

func lookupEnvOrDefault(key, value string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return value
}
