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
	"github.com/onsi/gomega"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func Job(t Test, namespace, name string) func(g gomega.Gomega) *batchv1.Job {
	return func(g gomega.Gomega) *batchv1.Job {
		job, err := t.Client().Core().BatchV1().Jobs(namespace).Get(t.Ctx(), name, metav1.GetOptions{})
		g.Expect(err).NotTo(gomega.HaveOccurred())
		return job
	}
}

func GetJob(t Test, namespace, name string) *batchv1.Job {
	t.T().Helper()
	return Job(t, namespace, name)(t)
}

func WriteJobLogs(t Test, namespace, name string) {
	t.T().Helper()

	job := GetJob(t, namespace, name)

	pods := GetPods(t, job.Namespace, metav1.ListOptions{
		LabelSelector: labels.FormatLabels(job.Spec.Selector.MatchLabels)},
	)

	if len(pods) == 0 {
		t.T().Errorf("Job %s/%s has no pods scheduled", job.Namespace, job.Name)
	} else {
		for i, pod := range pods {
			t.T().Logf("Retrieving Pod %s/%s logs", pod.Namespace, pod.Name)
			WriteToOutputDir(t, pod.Name, Log, GetPodLogs(t, &pods[i], corev1.PodLogOptions{}))
		}
	}
}
