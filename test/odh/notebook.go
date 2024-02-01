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

package odh

import (
	"bytes"

	gomega "github.com/onsi/gomega"
	. "github.com/project-codeflare/codeflare-common/support"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"

	imagev1 "github.com/openshift/api/image/v1"
)

const recommendedTagAnnotation = "opendatahub.io/workbench-image-recommended"

var notebookResource = schema.GroupVersionResource{Group: "kubeflow.org", Version: "v1", Resource: "notebooks"}

type NotebookProps struct {
	IngressDomain             string
	OpenShiftApiUrl           string
	KubernetesBearerToken     string
	Namespace                 string
	OpenDataHubNamespace      string
	ImageStreamName           string
	ImageStreamTag            string
	RayImage                  string
	NotebookConfigMapName     string
	NotebookConfigMapFileName string
	NotebookPVC               string
}

func createNotebook(test Test, namespace *corev1.Namespace, notebookToken, jupyterNotebookConfigMapName, jupyterNotebookConfigMapFileName string) {
	// Create PVC for Notebook
	notebookPVC := CreatePersistentVolumeClaim(test, namespace.Name, "10Gi", corev1.ReadWriteOnce)

	// Retrieve ImageStream tag for
	is := GetImageStream(test, GetOpenDataHubNamespace(), GetNotebookImageStreamName(test))
	recommendedTagName := getRecommendedImageStreamTag(test, is)

	// Read the Notebook CR from resources and perform replacements for custom values using go template
	notebookProps := NotebookProps{
		IngressDomain:             GetOpenShiftIngressDomain(test),
		OpenShiftApiUrl:           GetOpenShiftApiUrl(test),
		KubernetesBearerToken:     notebookToken,
		Namespace:                 namespace.Name,
		OpenDataHubNamespace:      GetOpenDataHubNamespace(),
		ImageStreamName:           GetNotebookImageStreamName(test),
		ImageStreamTag:            recommendedTagName,
		RayImage:                  GetRayImage(),
		NotebookConfigMapName:     jupyterNotebookConfigMapName,
		NotebookConfigMapFileName: jupyterNotebookConfigMapFileName,
		NotebookPVC:               notebookPVC.Name,
	}
	notebookTemplate, err := files.ReadFile("resources/custom-nb-small.yaml")
	test.Expect(err).NotTo(gomega.HaveOccurred())

	parsedNotebookTemplate := ParseTemplate(test, notebookTemplate, notebookProps)

	// Create Notebook CR
	notebookCR := &unstructured.Unstructured{}
	err = yaml.NewYAMLOrJSONDecoder(bytes.NewBuffer(parsedNotebookTemplate), 8192).Decode(notebookCR)
	test.Expect(err).NotTo(gomega.HaveOccurred())
	_, err = test.Client().Dynamic().Resource(notebookResource).Namespace(namespace.Name).Create(test.Ctx(), notebookCR, metav1.CreateOptions{})
	test.Expect(err).NotTo(gomega.HaveOccurred())
}

func getRecommendedImageStreamTag(test Test, is *imagev1.ImageStream) (tagName string) {
	for _, tag := range is.Spec.Tags {
		if tag.Annotations[recommendedTagAnnotation] == "true" {
			return tag.Name
		}
	}
	test.T().Fatalf("tag with annotation '%s' not found in ImageStream %s", recommendedTagAnnotation, is.Name)
	return
}
