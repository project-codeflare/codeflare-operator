package controllers

import (
	"os"

	"github.com/go-logr/logr"
	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/validation/field"
	v1 "k8s.io/client-go/applyconfigurations/meta/v1"
	networkingv1ac "k8s.io/client-go/applyconfigurations/networking/v1"

	routeapply "github.com/openshift/client-go/route/applyconfigurations/route/v1"
)

var (
	CertGeneratorImage = getEnv("CERT_GENERATOR_IMAGE", "registry.redhat.io/ubi9@sha256:770cf07083e1c85ae69c25181a205b7cdef63c11b794c89b3b487d4670b4c328")
	OAuthProxyImage    = getEnv("OAUTH_PROXY_IMAGE", "registry.redhat.io/openshift4/ose-oauth-proxy@sha256:1ea6a01bf3e63cdcf125c6064cbd4a4a270deaf0f157b3eabb78f60556840366")
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func serviceNameFromCluster(cluster *rayv1.RayCluster) string {
	return cluster.Name + "-head-svc"
}

func desiredRayClientRoute(cluster *rayv1.RayCluster) *routeapply.RouteApplyConfiguration {
	return routeapply.Route(rayClientNameFromCluster(cluster), cluster.Namespace).
		WithLabels(map[string]string{RayClusterNameLabel: cluster.Name}).
		WithSpec(routeapply.RouteSpec().
			WithTo(routeapply.RouteTargetReference().WithKind("Service").WithName(serviceNameFromCluster(cluster)).WithWeight(100)).
			WithPort(routeapply.RoutePort().WithTargetPort(intstr.FromString("client"))).
			WithTLS(routeapply.TLSConfig().WithTermination("passthrough")),
		).
		WithOwnerReferences(
			v1.OwnerReference().WithUID(cluster.UID).WithName(cluster.Name).WithKind(cluster.Kind).WithAPIVersion(cluster.APIVersion).WithController(true),
		)
}

func desiredRayClientIngress(cluster *rayv1.RayCluster, ingressHost string) *networkingv1ac.IngressApplyConfiguration {
	return networkingv1ac.Ingress(rayClientNameFromCluster(cluster), cluster.Namespace).
		WithLabels(map[string]string{RayClusterNameLabel: cluster.Name}).
		WithAnnotations(map[string]string{
			"nginx.ingress.kubernetes.io/rewrite-target":  "/",
			"nginx.ingress.kubernetes.io/ssl-redirect":    "true",
			"nginx.ingress.kubernetes.io/ssl-passthrough": "true",
		}).
		WithOwnerReferences(v1.OwnerReference().
			WithAPIVersion(cluster.APIVersion).
			WithKind(cluster.Kind).
			WithName(cluster.Name).
			WithUID(cluster.UID).
			WithController(true)).
		WithSpec(networkingv1ac.IngressSpec().
			WithIngressClassName("nginx").
			WithRules(networkingv1ac.IngressRule().
				WithHost(ingressHost).
				WithHTTP(networkingv1ac.HTTPIngressRuleValue().
					WithPaths(networkingv1ac.HTTPIngressPath().
						WithPath("/").
						WithPathType(networkingv1.PathTypeImplementationSpecific).
						WithBackend(networkingv1ac.IngressBackend().
							WithService(networkingv1ac.IngressServiceBackend().
								WithName(serviceNameFromCluster(cluster)).
								WithPort(networkingv1ac.ServiceBackendPort().
									WithNumber(10001),
								),
							),
						),
					),
				),
			),
		)
}

func desiredClusterIngress(cluster *rayv1.RayCluster, ingressHost string) *networkingv1ac.IngressApplyConfiguration {
	return networkingv1ac.Ingress(dashboardNameFromCluster(cluster), cluster.Namespace).
		WithLabels(map[string]string{RayClusterNameLabel: cluster.Name}).
		WithOwnerReferences(v1.OwnerReference().
			WithAPIVersion(cluster.APIVersion).
			WithKind(cluster.Kind).
			WithName(cluster.Name).
			WithUID(cluster.UID).
			WithController(true)).
		WithSpec(networkingv1ac.IngressSpec().
			WithRules(networkingv1ac.IngressRule().
				WithHost(ingressHost). // Full Hostname
				WithHTTP(networkingv1ac.HTTPIngressRuleValue().
					WithPaths(networkingv1ac.HTTPIngressPath().
						WithPath("/").
						WithPathType(networkingv1.PathTypePrefix).
						WithBackend(networkingv1ac.IngressBackend().
							WithService(networkingv1ac.IngressServiceBackend().
								WithName(serviceNameFromCluster(cluster)).
								WithPort(networkingv1ac.ServiceBackendPort().
									WithName(ingressServicePortName),
								),
							),
						),
					),
				),
			),
		)
}

type compare[T any] func(T, T) bool

func upsert[T any](items []T, item T, predicate compare[T]) []T {
	for i, t := range items {
		if predicate(t, item) {
			items[i] = item
			return items
		}
	}
	return append(items, item)
}

func contains[T any](items []T, item T, predicate compare[T], path *field.Path, msg string) *field.Error {
	for _, t := range items {
		if predicate(t, item) {
			if equality.Semantic.DeepDerivative(item, t) {
				return nil
			}
			return field.Invalid(path, t, msg)
		}
	}
	return field.Required(path, msg)
}

var byContainerName = compare[corev1.Container](
	func(c1, c2 corev1.Container) bool {
		return c1.Name == c2.Name
	})

func withContainerName(name string) compare[corev1.Container] {
	return func(c1, c2 corev1.Container) bool {
		return c1.Name == name
	}
}

var byVolumeName = compare[corev1.Volume](
	func(v1, v2 corev1.Volume) bool {
		return v1.Name == v2.Name
	})

func withVolumeName(name string) compare[corev1.Volume] {
	return func(v1, v2 corev1.Volume) bool {
		return v1.Name == name
	}
}

var byVolumeMountName = compare[corev1.VolumeMount](
	func(v1, v2 corev1.VolumeMount) bool {
		return v1.Name == v2.Name
	})

var byEnvVarName = compare[corev1.EnvVar](
	func(e1, e2 corev1.EnvVar) bool {
		return e1.Name == e2.Name
	})

func withEnvVarName(name string) compare[corev1.EnvVar] {
	return func(e1, e2 corev1.EnvVar) bool {
		return e1.Name == name
	}
}

// logSink implements a log sink with an error log filter
type logSink struct {
	sink logr.LogSink
}

func (l logSink) Init(info logr.RuntimeInfo) {
	l.sink.Init(info)
}

func (l logSink) Enabled(level int) bool {
	return l.sink.Enabled(level)
}
func (l logSink) Info(level int, msg string, keysAndValues ...any) {
	l.sink.Info(level, msg, keysAndValues...)
}

func (l logSink) Error(err error, msg string, keysAndValues ...any) {
	// downgrade StatusReasonConflict errors to debug messages
	if errors.IsConflict(err) {
		l.sink.Info(1, msg, append(keysAndValues, "error", err.Error())...)
	} else {
		l.sink.Error(err, msg, keysAndValues...)
	}
}

func (l logSink) WithValues(keysAndValues ...any) logr.LogSink {
	return logSink{l.sink.WithValues(keysAndValues...)}
}

func (l logSink) WithName(name string) logr.LogSink {
	return logSink{l.sink.WithName(name)}
}

// FilteredLogger returns a copy of the logger with an error log filter
func FilteredLogger(logger logr.Logger) logr.Logger {
	return logger.WithSink(logSink{logger.GetSink()})
}
