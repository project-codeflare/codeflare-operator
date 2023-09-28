package e2e

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
	mcadv1beta1 "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/apis/controller/v1beta1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	. "github.com/project-codeflare/codeflare-operator/test/support"
)

func TestInstascaleMachinePool(t *testing.T) {

	test := With(t)
	test.T().Parallel()

	namespace := test.NewTestNamespace()

	// Test configuration
	config, err := TestConfig(test, namespace.Name)
	test.Expect(err).To(BeNil())

	//create OCM connection
	connection, err := CreateConnection(test)
	test.Expect(err).To(BeNil())

	defer connection.Close()

	// check existing cluster machine pool resources
	// look for machine pool with aw name - expect not to find it
	foundMachinePool, err := CheckMachinePools(connection, TestName)
	test.Expect(err).NotTo(HaveOccurred())
	test.Expect(foundMachinePool).To(BeFalse())

	// Setup batch job and AppWrapper
	job, aw, err := JobAppwrapperSetup(test, namespace, config)
	test.Expect(err).To(BeNil())

	// time.Sleep is used twice throughout the test, each for 30 seconds. Can look into using sync package waitGroup instead if that makes more sense
	// wait for required resources to scale up before checking them again
	time.Sleep(TestTimeoutMedium)

	// look for machine pool with aw name - expect to find it
	foundMachinePool, err = CheckMachinePools(connection, TestName)
	test.Expect(err).NotTo(HaveOccurred())
	test.Expect(foundMachinePool).To(BeTrue())

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
	time.Sleep(TestTimeoutMedium)

	// look for machine pool with aw name - expect not to find it
	foundMachinePool, err = CheckMachinePools(connection, TestName)
	test.Expect(err).NotTo(HaveOccurred())
	test.Expect(foundMachinePool).To(BeFalse())

}
