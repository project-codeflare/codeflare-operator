package e2e

import (
	"sync"
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

var (
	machinePoolsExist     bool
	numInitialNodePools   int
	numInitialMachineSets int
	wg                    = &sync.WaitGroup{}
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

	machinePoolsExist = true
	// check existing cluster resources
	numInitialMachinePools, err := MachinePoolsCount(connection)
	if err != nil {
		test.T().Errorf("Unable to count machine pools - Error : %v", err)
	}

	if numInitialMachinePools == 0 {
		machinePoolsExist = false
		numInitialNodePools, err = NodePoolsCount(connection)
		if err != nil {
			test.T().Errorf("Unable to count node pools - Error : %v", err)
		}
		if numInitialNodePools == 0 {
			numInitialMachineSets, err = MachineSetsCount()
			if err != nil {
				test.T().Errorf("Unable to count machine sets - Error : %v", err)
			}
		}
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
							Name:    "job",
							Image:   GetPyTorchImage(),
							Env: []corev1.EnvVar{
								corev1.EnvVar{Name: "PYTHONUSERBASE", Value: "/test2"},
							},
							Command: []string{"/bin/sh", "-c", "pip install -r /test/requirements.txt && torchrun /test/mnist.py"},
							Args: []string{"$PYTHONUSERBASE"},
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
						GenericTemplate: Raw(test, job),
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

	// wait for required resources to be created before checking them again
	time.Sleep(TestTimeoutShort)
	if !machinePoolsExist {
		numNodePools, err := NodePoolsCount(connection)
		if err != nil {
			test.T().Errorf("Unable to count node pools - Error : %v", err)
		}
		test.Expect(numNodePools).To(BeNumerically(">", numInitialNodePools))
		test.T().Logf("number of node pools increased from %d to %d", numInitialNodePools, numNodePools)

	} else if machinePoolsExist {
		numMachinePools, err := MachinePoolsCount(connection)
		if err != nil {
			test.T().Errorf("Unable to count machine pools - Error : %v", err)
		}
		test.Expect(numMachinePools).To(BeNumerically(">", numInitialMachinePools))
		test.T().Logf("number of machine pools increased from %d to %d", numInitialMachinePools, numMachinePools)
	} else {
		numMachineSets, err := MachineSetsCount()
		if err != nil {
			test.T().Errorf("Unable to count machine sets - Error : %v", err)
		}
		test.Expect(numMachineSets).To(BeNumerically(">", numInitialMachineSets))
		test.T().Logf("number of machine sets increased from %d to %d", numInitialMachineSets, numMachineSets)
	}
	
	test.T().Logf("Waiting for Job %s/%s to complete", job.Namespace, job.Name)
	test.Eventually(Job(test, job.Namespace, job.Name), TestTimeoutLong).Should(
		Or(
			WithTransform(ConditionStatus(batchv1.JobComplete), Equal(corev1.ConditionTrue)),
			WithTransform(ConditionStatus(batchv1.JobFailed), Equal(corev1.ConditionTrue)),
		))

	// Assert the job has completed successfully
	test.Expect(GetJob(test, job.Namespace, job.Name)).
		To(WithTransform(ConditionStatus(batchv1.JobComplete), Equal(corev1.ConditionTrue)))

	// AppWrapper not being updated to complete once job is finished

	time.Sleep(TestTimeoutMedium)
	if !machinePoolsExist {
		numNodePoolsFinal, err := NodePoolsCount(connection)
		if err != nil {
			test.T().Errorf("Unable to count node pools - Error : %v", err)
		}
		test.Expect(numNodePoolsFinal).To(BeNumerically("==", numInitialNodePools))
		test.T().Logf("number of machine pools decreased")

	} else if machinePoolsExist {
		numMachinePoolsFinal, err := MachinePoolsCount(connection)
		if err != nil {
			test.T().Errorf("Unable to count machine pools - Error : %v", err)
		}
		test.Expect(numMachinePoolsFinal).To(BeNumerically("==", numInitialMachinePools))
		test.T().Logf("number of machine pools decreased")
	} else {
		numMachineSetsFinal, err := MachineSetsCount()
		if err != nil {
			test.T().Errorf("Unable to count machine sets - Error : %v", err)
		}
		test.Expect(numMachineSetsFinal).To(BeNumerically("==", numInitialMachineSets))
		test.T().Logf("number of machine sets decreased")
	}
}
