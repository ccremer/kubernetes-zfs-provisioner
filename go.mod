module github.com/ccremer/kubernetes-zfs-provisioner

go 1.16

require (
	github.com/knadh/koanf v0.16.0
	github.com/mistifyio/go-zfs v2.1.1+incompatible
	github.com/stretchr/testify v1.7.0
	k8s.io/api v0.19.1
	k8s.io/apimachinery v0.19.1
	k8s.io/client-go v0.19.1
	k8s.io/klog/v2 v2.8.0
	sigs.k8s.io/sig-storage-lib-external-provisioner/v6 v6.3.0
)
