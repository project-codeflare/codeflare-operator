module github.com/project-codeflare/codeflare-operator

go 1.22.4

require (
	github.com/go-logr/logr v1.4.2
	github.com/onsi/ginkgo/v2 v2.19.0
	github.com/onsi/gomega v1.33.1
	github.com/open-policy-agent/cert-controller v0.10.1
	github.com/opendatahub-io/opendatahub-operator/v2 v2.10.0
	github.com/openshift/api v0.0.0-20230823114715-5fdd7511b790
	github.com/openshift/client-go v0.0.0-20221019143426-16aed247da5c
	github.com/project-codeflare/appwrapper v0.30.0
	github.com/project-codeflare/codeflare-common v0.0.0-20241216183607-222395d38924
	github.com/ray-project/kuberay/ray-operator v1.2.1
	go.uber.org/zap v1.27.0
	golang.org/x/exp v0.0.0-20240613232115-7f521ea00fb8
	k8s.io/api v0.30.2
	k8s.io/apiextensions-apiserver v0.29.6
	k8s.io/apimachinery v0.30.2
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/component-base v0.29.6
	k8s.io/klog/v2 v2.130.1
	k8s.io/utils v0.0.0-20240502163921-fe8a2dddb1d0
	sigs.k8s.io/controller-runtime v0.17.5
	sigs.k8s.io/kueue v0.8.3
	sigs.k8s.io/yaml v1.4.0
)

replace k8s.io/client-go => k8s.io/client-go v0.29.2

replace sigs.k8s.io/custom-metrics-apiserver => sigs.k8s.io/custom-metrics-apiserver v1.25.1-0.20230306170449-63d8c93851f3

replace go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.44.0

replace github.com/jackc/pgx/v4 => github.com/jackc/pgx/v5 v5.5.4

// This replace directive deal with the backlevel ODH kueue version
replace sigs.k8s.io/kueue v0.8.3 => github.com/opendatahub-io/kueue v0.8.3

// This replace directive deal with the unaligned Ray operator version transitively pulled by Kueue 0.8.3
replace github.com/ray-project/kuberay/ray-operator v1.2.1 => github.com/ray-project/kuberay/ray-operator v1.1.1

require (
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/distribution/reference v0.5.0 // indirect
	github.com/emicklei/go-restful/v3 v3.12.1 // indirect
	github.com/evanphx/json-patch v5.9.0+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.9.0 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/go-logr/zapr v1.3.0 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.0 // indirect
	github.com/golang/glog v1.1.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/gnostic-models v0.6.8 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20240424215950-a892ee059fd6 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/css v1.0.0 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/jackc/pgx/v5 v5.5.5 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/kubeflow/training-operator v1.7.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/microcosm-cc/bluemonday v1.0.18 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mwitkow/go-conntrack v0.0.0-20190716064945-2f068394615f // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/openshift-online/ocm-sdk-go v0.1.411 // indirect
	github.com/openshift/custom-resource-status v1.1.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.20.4 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.57.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.28.0 // indirect
	golang.org/x/oauth2 v0.21.0 // indirect
	golang.org/x/sys v0.23.0 // indirect
	golang.org/x/term v0.23.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.org/x/tools v0.24.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.4.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiserver v0.29.6 // indirect
	k8s.io/kube-openapi v0.0.0-20240620174524-b456828f718b // indirect
	sigs.k8s.io/jobset v0.5.2 // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.1 // indirect
)
