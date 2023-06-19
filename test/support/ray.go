package support

import (
	"github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rayv1alpha1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1alpha1"
)

const RayJobDefaultClusterSelectorKey = "ray.io/cluster"

func RayJob(t Test, namespace *corev1.Namespace, name string) func(g gomega.Gomega) *rayv1alpha1.RayJob {
	return func(g gomega.Gomega) *rayv1alpha1.RayJob {
		job, err := t.Client().Ray().RayV1alpha1().RayJobs(namespace.Name).Get(t.Ctx(), name, metav1.GetOptions{})
		g.Expect(err).NotTo(gomega.HaveOccurred())
		return job
	}
}

func RayJobStatus(job *rayv1alpha1.RayJob) rayv1alpha1.JobStatus {
	return job.Status.JobStatus
}
