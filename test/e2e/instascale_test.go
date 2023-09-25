package e2e

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
	. "github.com/project-codeflare/codeflare-operator/test/support"
	mcadv1beta1 "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/apis/controller/v1beta1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInstascale(t *testing.T) {

	test := With(t)
	test.T().Parallel()

	namespace := test.NewTestNamespace()

	// Test configuration
	config := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mnist-mcad",
			Namespace: namespace.Name,
		},
		BinaryData: map[string][]byte{
			// pip requirements
			"requirements.txt": ReadFile(test, "mnist_pip_requirements.txt"),
			// MNIST training script
			"mnist.py": ReadFile(test, "mnist.py"),
		},
		Immutable: Ptr(true),
	}
	config, err := test.Client().Core().CoreV1().ConfigMaps(namespace.Name).Create(test.Ctx(), config, metav1.CreateOptions{})
	test.Expect(err).NotTo(HaveOccurred())
	test.T().Logf("Created ConfigMap %s/%s successfully", config.Namespace, config.Name)

	// create OCM connection
	instascaleOCMSecret, err := test.Client().Core().CoreV1().Secrets("default").Get(test.Ctx(), "instascale-ocm-secret", metav1.GetOptions{})
	if err != nil {
		test.T().Errorf("unable to retrieve instascale-ocm-secret - Error : %v", err)
	}
	test.Expect(err).NotTo(HaveOccurred())
	ocmToken := string(instascaleOCMSecret.Data["token"])
	test.T().Logf("Retrieved Secret %s successfully", instascaleOCMSecret.Name)

	connection, err := CreateOCMConnection(ocmToken)
	if err != nil {
		test.T().Errorf("Unable to create ocm connection - Error : %v", err)
	}
	defer connection.Close()

	// check existing cluster resources
	machinePoolsExist, err := MachinePoolsExist(connection)
	test.Expect(err).NotTo(HaveOccurred())
	nodePoolsExist, err := NodePoolsExist(connection)
	test.Expect(err).NotTo(HaveOccurred())

	if machinePoolsExist {
		// look for machine pool with aw name - expect not to find it
		foundMachinePool, err := CheckMachinePools(connection, TestName)
		test.Expect(err).NotTo(HaveOccurred())
		test.Expect(foundMachinePool).To(BeFalse())
	} else if nodePoolsExist {
		// look for node pool with aw name - expect not to find it
		foundNodePool, err := CheckNodePools(connection, TestName)
		test.Expect(err).NotTo(HaveOccurred())
		test.Expect(foundNodePool).To(BeFalse())
	} else {
		foundMachineSet, err := CheckMachineSets(TestName)
		test.Expect(err).NotTo(HaveOccurred())
		test.Expect(foundMachineSet).To(BeFalse())
	}

	// Batch Job
	job := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			APIVersion: batchv1.SchemeGroupVersion.String(),
			Kind:       "Job",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mnist",
			Namespace: namespace.Name,
		},
		Spec: batchv1.JobSpec{
			Completions: Ptr(int32(1)),
			Parallelism: Ptr(int32(1)),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "job",
							Image: GetPyTorchImage(),
							Env: []corev1.EnvVar{
								corev1.EnvVar{Name: "PYTHONUSERBASE", Value: "/test2"},
							},
							Command: []string{"/bin/sh", "-c", "pip install -r /test/requirements.txt && torchrun /test/mnist.py"},
							Args:    []string{"$PYTHONUSERBASE"},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "test",
									MountPath: "/test",
								},
								{
									Name:      "test2",
									MountPath: "/test2",
								},
							},
							WorkingDir: "/test2",
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "test",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: config.Name,
									},
								},
							},
						},
						{
							Name: "test2",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}

	// create an appwrapper
	aw := &mcadv1beta1.AppWrapper{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instascale",
			Namespace: namespace.Name,
			Labels: map[string]string{
				"orderedinstance": "m5.xlarge_g4dn.xlarge",
			},
		},
		Spec: mcadv1beta1.AppWrapperSpec{
			AggrResources: mcadv1beta1.AppWrapperResourceList{
				GenericItems: []mcadv1beta1.AppWrapperGenericResource{
					{
						CustomPodResources: []mcadv1beta1.CustomPodResourceTemplate{
							{
								Replicas: 1,
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("250m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("1G"),
								},
							},
							{
								Replicas: 1,
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("250m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("1G"),
								},
							},
						},
						GenericTemplate:  Raw(test, job),
						CompletionStatus: "Complete",
					},
				},
			},
		},
	}

	_, err = test.Client().MCAD().WorkloadV1beta1().AppWrappers(namespace.Name).Create(test.Ctx(), aw, metav1.CreateOptions{})
	test.Expect(err).NotTo(HaveOccurred())
	test.T().Logf("AppWrapper created successfully %s/%s", aw.Namespace, aw.Name)

	test.Eventually(AppWrapper(test, namespace, aw.Name), TestTimeoutShort).
		Should(WithTransform(AppWrapperState, Equal(mcadv1beta1.AppWrapperStateActive)))

	// time.Sleep is used twice throughout the test, each for 30 seconds. Can look into using sync package waitGroup instead if that makes more sense
	// wait for required resources to scale up before checking them again
	time.Sleep(TestTimeoutThirtySeconds)

	if machinePoolsExist {
		// look for machine pool with aw name - expect to find it
		foundMachinePool, err := CheckMachinePools(connection, TestName)
		test.Expect(err).NotTo(HaveOccurred())
		test.Expect(foundMachinePool).To(BeTrue())
	} else if nodePoolsExist {
		// look for node pool with aw name - expect to find it
		foundNodePool, err := CheckNodePools(connection, TestName)
		test.Expect(err).NotTo(HaveOccurred())
		test.Expect(foundNodePool).To(BeTrue())
	} else {
		foundMachineSet, err := CheckMachineSets(TestName)
		test.Expect(err).NotTo(HaveOccurred())
		test.Expect(foundMachineSet).To(BeTrue())
	}

	// Assert that the job has completed
	test.T().Logf("Waiting for Job %s/%s to complete", job.Namespace, job.Name)
	test.Eventually(Job(test, job.Namespace, job.Name), TestTimeoutLong).Should(
		Or(
			WithTransform(ConditionStatus(batchv1.JobComplete), Equal(corev1.ConditionTrue)),
			WithTransform(ConditionStatus(batchv1.JobFailed), Equal(corev1.ConditionTrue)),
		))

	// Assert the job has completed successfully
	test.Expect(GetJob(test, job.Namespace, job.Name)).
		To(WithTransform(ConditionStatus(batchv1.JobComplete), Equal(corev1.ConditionTrue)))

	test.Eventually(AppWrapper(test, namespace, aw.Name), TestTimeoutShort).
		Should(WithTransform(AppWrapperState, Equal(mcadv1beta1.AppWrapperStateCompleted)))

	// allow time for the resources to scale down before checking them again
	time.Sleep(TestTimeoutThirtySeconds)

	if machinePoolsExist {
		// look for machine pool with aw name - expect to find it
		foundMachinePool, err := CheckMachinePools(connection, TestName)
		test.Expect(err).NotTo(HaveOccurred())
		test.Expect(foundMachinePool).To(BeFalse())
	} else if nodePoolsExist {
		// look for node pool with aw name - expect to find it
		foundNodePool, err := CheckNodePools(connection, TestName)
		test.Expect(err).NotTo(HaveOccurred())
		test.Expect(foundNodePool).To(BeFalse())
	} else {
		foundMachineSet, err := CheckMachineSets(TestName)
		test.Expect(err).NotTo(HaveOccurred())
		test.Expect(foundMachineSet).To(BeFalse())
	}
}
