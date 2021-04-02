module github.com/ccremer/kubernetes-zfs-provisioner

go 1.14

require (
	github.com/knadh/koanf v0.15.0
	github.com/miekg/dns v1.1.31 // indirect
	github.com/mistifyio/go-zfs v2.1.1+incompatible
	github.com/prometheus/client_golang v1.7.1 // indirect
	github.com/stretchr/testify v1.7.0
	k8s.io/api v0.19.1
	k8s.io/apimachinery v0.19.1
	k8s.io/client-go v0.19.1
	k8s.io/klog v1.0.0
	sigs.k8s.io/sig-storage-lib-external-provisioner/v6 v6.3.0
)
