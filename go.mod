module github.com/kaloom/kubernetes-podagent

go 1.18

require (
	github.com/blang/semver v3.5.1+incompatible
	github.com/containernetworking/cni v0.8.1
	github.com/docker/docker v20.10.7+incompatible
	github.com/golang/glog v1.0.0
	github.com/kaloom/kubernetes-common v0.1.5
	github.com/pkg/errors v0.9.1
	google.golang.org/grpc v1.40.0
	k8s.io/api v0.23.15
	k8s.io/apimachinery v0.23.15
	k8s.io/client-go v0.23.15
	k8s.io/cri-api v0.0.0
	k8s.io/kubernetes v1.23.15
	k8s.io/utils v0.0.0-20211116205334-6203023598ed
)

require (
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/Microsoft/go-winio v0.4.17 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/containerd/containerd v1.4.11 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/go-logr/logr v1.2.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.5.5 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/imdario/mergo v0.3.5 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/moby/term v0.0.0-20210610120745-9d4ed1856297 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.2 // indirect
	github.com/prometheus/client_golang v1.11.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.28.0 // indirect
	github.com/prometheus/procfs v0.6.0 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/net v0.0.0-20211209124913-491a49abca63 // indirect
	golang.org/x/oauth2 v0.0.0-20210819190943-2bc19b11175f // indirect
	golang.org/x/sys v0.0.0-20210831042530-f4d43177bf5e // indirect
	golang.org/x/term v0.0.0-20210615171337-6886f2dfbf5b // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20210831024726-fe130286e0e2 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/component-base v0.23.15 // indirect
	k8s.io/klog/v2 v2.30.0 // indirect
	k8s.io/kube-openapi v0.0.0-20211115234752-e816edb12b65 // indirect
	sigs.k8s.io/json v0.0.0-20211020170558-c049b76a60c6 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)

replace k8s.io/api => k8s.io/api v0.23.15

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.23.15

replace k8s.io/apimachinery => k8s.io/apimachinery v0.23.16-rc.0

replace k8s.io/apiserver => k8s.io/apiserver v0.23.15

replace k8s.io/cli-runtime => k8s.io/cli-runtime v0.23.15

replace k8s.io/client-go => k8s.io/client-go v0.23.15

replace k8s.io/cloud-provider => k8s.io/cloud-provider v0.23.15

replace k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.23.15

replace k8s.io/code-generator => k8s.io/code-generator v0.23.16-rc.0

replace k8s.io/component-base => k8s.io/component-base v0.23.15

replace k8s.io/component-helpers => k8s.io/component-helpers v0.23.15

replace k8s.io/controller-manager => k8s.io/controller-manager v0.23.15

replace k8s.io/cri-api => k8s.io/cri-api v0.23.16-rc.0

replace k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.23.15

replace k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.23.15

replace k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.23.15

replace k8s.io/kube-proxy => k8s.io/kube-proxy v0.23.15

replace k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.23.15

replace k8s.io/kubectl => k8s.io/kubectl v0.23.15

replace k8s.io/kubelet => k8s.io/kubelet v0.23.15

replace k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.23.15

replace k8s.io/metrics => k8s.io/metrics v0.23.15

replace k8s.io/mount-utils => k8s.io/mount-utils v0.23.16-rc.0

replace k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.23.15

replace k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.23.15

replace k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.23.15

replace k8s.io/sample-controller => k8s.io/sample-controller v0.23.15
