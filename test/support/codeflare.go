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

// The environment variables hereafter can be used to change the components
// used for testing.
const (
	CodeFlareTestSdkVersion = "CODEFLARE_TEST_SDK_VERSION"
	CodeFlareTestRayVersion = "CODEFLARE_TEST_RAY_VERSION"
	CodeFlareTestRayImage   = "CODEFLARE_TEST_RAY_IMAGE"
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

func lookupEnvOrDefault(key, value string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return value
}
