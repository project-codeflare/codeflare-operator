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
	"encoding/json"

	"github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rayv1alpha1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1alpha1"
)

const RayJobDefaultClusterSelectorKey = "ray.io/cluster"

func RayJob(t Test, namespace, name string) func(g gomega.Gomega) *rayv1alpha1.RayJob {
	return func(g gomega.Gomega) *rayv1alpha1.RayJob {
		job, err := t.Client().Ray().RayV1alpha1().RayJobs(namespace).Get(t.Ctx(), name, metav1.GetOptions{})
		g.Expect(err).NotTo(gomega.HaveOccurred())
		return job
	}
}

func GetRayJob(t Test, namespace, name string) *rayv1alpha1.RayJob {
	t.T().Helper()
	return RayJob(t, namespace, name)(t)
}

func RayJobStatus(job *rayv1alpha1.RayJob) rayv1alpha1.JobStatus {
	return job.Status.JobStatus
}

func GetRayJobLogs(t Test, namespace, name string) []byte {
	t.T().Helper()

	job := GetRayJob(t, namespace, name)

	response := t.Client().Core().CoreV1().RESTClient().
		Get().
		AbsPath("/api/v1/namespaces", job.Namespace, "services", "http:"+job.Status.RayClusterName+"-head-svc:dashboard", "proxy", "api", "jobs", job.Status.JobId, "logs").
		Do(t.Ctx())
	t.Expect(response.Error()).NotTo(gomega.HaveOccurred())

	body := map[string]string{}
	bytes, _ := response.Raw()
	t.Expect(json.Unmarshal(bytes, &body)).To(gomega.Succeed())
	t.Expect(body).To(gomega.HaveKey("logs"))

	return []byte(body["logs"])
}
