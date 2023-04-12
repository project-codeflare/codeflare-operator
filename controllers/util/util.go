package util

import (
	"fmt"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

func notEqualMsg(value string) {
	print(fmt.Sprintf("%s are not equal.", value))
}

func ConfigMapsAreEqual(expected corev1.ConfigMap, actual corev1.ConfigMap) bool {
	if expected.Name != actual.Name {
		notEqualMsg("Configmap Names")
		return false
	}

	if !reflect.DeepEqual(expected.Data, actual.Data) {
		notEqualMsg("Configmap Data values")
		return false
	}
	return true
}

//func RoleBindingsAreEqual(rb1 k8srbacv1.RoleBinding, rb2 k8srbacv1.RoleBinding) bool {
//
//	if !reflect.DeepEqual(rb1.ObjectMeta.Labels, rb2.ObjectMeta.Labels) {
//		notEqualMsg("Rolebinding labels")
//		return false
//	}
//	if !reflect.DeepEqual(rb1.Subjects, rb2.Subjects) {
//		notEqualMsg("Rolebinding subjects")
//		return false
//	}
//	if !reflect.DeepEqual(rb1.RoleRef, rb2.RoleRef) {
//		notEqualMsg("Rolebinding role references")
//		return false
//	}
//	return true
//}

func ServiceAccountsAreEqual(sa1 corev1.ServiceAccount, sa2 corev1.ServiceAccount) bool {
	if !reflect.DeepEqual(sa1.ObjectMeta.Labels, sa2.ObjectMeta.Labels) {
		notEqualMsg("ServiceAccount labels")
		return false
	}
	if !reflect.DeepEqual(sa1.Name, sa2.Name) {
		notEqualMsg("ServiceAccount names")
		return false
	}
	if !reflect.DeepEqual(sa1.Namespace, sa2.Namespace) {
		notEqualMsg("ServiceAccount namespaces")
		return false
	}
	return true
}

func ServicesAreEqual(service1 corev1.Service, service2 corev1.Service) bool {
	if !reflect.DeepEqual(service1.ObjectMeta.Labels, service2.ObjectMeta.Labels) {
		notEqualMsg("Service labels")
		return false
	}
	if !reflect.DeepEqual(service1.Name, service2.Name) {
		notEqualMsg("Service Names")
		return false
	}
	if !reflect.DeepEqual(service1.Namespace, service2.Namespace) {
		notEqualMsg("Service namespaces")
		return false
	}
	if !reflect.DeepEqual(service1.Spec.Selector, service2.Spec.Selector) {
		notEqualMsg("Service Selectors")
		return false
	}
	if !reflect.DeepEqual(service1.Spec.Ports, service2.Spec.Ports) {
		notEqualMsg("Service Ports")
		return false
	}
	return true
}

func DeploymentsAreEqual(dp1 appsv1.Deployment, dp2 appsv1.Deployment) bool {

	if !reflect.DeepEqual(dp1.ObjectMeta.Labels, dp2.ObjectMeta.Labels) {
		notEqualMsg("labels")
		return false
	}

	if !reflect.DeepEqual(dp1.Spec.Selector, dp2.Spec.Selector) {
		notEqualMsg("selector")
		return false
	}

	if !reflect.DeepEqual(dp1.Spec.Template.ObjectMeta, dp2.Spec.Template.ObjectMeta) {
		notEqualMsg("Object MetaData")
		return false
	}

	if !reflect.DeepEqual(dp1.Spec.Template.Spec.Volumes, dp2.Spec.Template.Spec.Volumes) {
		notEqualMsg("Volumes")
		return false
	}

	if len(dp1.Spec.Template.Spec.Containers) != len(dp2.Spec.Template.Spec.Containers) {
		notEqualMsg("Containers")
		return false
	}
	for i := range dp1.Spec.Template.Spec.Containers {
		c1 := dp1.Spec.Template.Spec.Containers[i]
		c2 := dp2.Spec.Template.Spec.Containers[i]
		if !reflect.DeepEqual(c1.Env, c2.Env) {
			notEqualMsg("Container Env")
			return false
		}
		if !reflect.DeepEqual(c1.Ports, c2.Ports) {
			notEqualMsg("Container Ports")
			return false
		}
		if !reflect.DeepEqual(c1.Resources, c2.Resources) {
			notEqualMsg("Container Resources")
			return false
		}
		if !reflect.DeepEqual(c1.VolumeMounts, c2.VolumeMounts) {
			notEqualMsg("Container VolumeMounts")
			return false
		}
		if !reflect.DeepEqual(c1.Args, c2.Args) {
			notEqualMsg("Container Args")
			return false
		}
		if c1.Name != c2.Name {
			notEqualMsg("Container Name")
			return false
		}
		if c1.Image != c2.Image {
			notEqualMsg("Container Image")
			return false
		}
	}

	return true
}

func ClusterRolesAreEqual(cr1 rbacv1.ClusterRole, cr2 rbacv1.ClusterRole) bool {
	if !reflect.DeepEqual(cr1.Rules, cr2.Rules) {
		notEqualMsg("rules")
		return false
	}
	if !reflect.DeepEqual(cr1.ObjectMeta.Name, cr2.ObjectMeta.Name) {
		notEqualMsg("name")
		return false
	}
	return true
}

func ClusterRoleBindingsAreEqual(crb1 rbacv1.ClusterRoleBinding, crb2 rbacv1.ClusterRoleBinding) bool {
	if !reflect.DeepEqual(crb1.Subjects, crb2.Subjects) {
		notEqualMsg("subjects")
		return false
	}
	if !reflect.DeepEqual(crb1.RoleRef, crb2.RoleRef) {
		notEqualMsg("roleRef")
		return false
	}
	if !reflect.DeepEqual(crb1.ObjectMeta.Name, crb2.ObjectMeta.Name) {
		notEqualMsg("name")
		return false
	}
	return true
}

func IsDeploymentReady(deployment *appsv1.Deployment) bool {
	for _, condition := range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentAvailable && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}
