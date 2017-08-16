package provisioner

import (
	"os"
	"testing"

	"github.com/kubernetes-incubator/external-storage/lib/controller"
	zfs "github.com/mistifyio/go-zfs"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
)

func TestProvision(t *testing.T) {
	parent, _ := zfs.GetDataset("test/volumes")
	p := NewZFSProvisioner(parent, "", "127.0.0.1", "", "Delete")

	options := controller.VolumeOptions{
		PersistentVolumeReclaimPolicy: v1.PersistentVolumeReclaimDelete,
		PVName: "pv-testcreate",
		PVC:    newClaim(resource.MustParse("1G"), []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce, v1.ReadOnlyMany}, nil),
	}
	pv, err := p.Provision(options)

	assert.NoError(t, err, "Provision should not return an error")
	_, err = os.Stat(pv.Spec.PersistentVolumeSource.NFS.Path)
	assert.NoError(t, err, "The volume should exist on disk")
}

func newClaim(capacity resource.Quantity, accessmodes []v1.PersistentVolumeAccessMode, selector *metav1.LabelSelector) *v1.PersistentVolumeClaim {
	claim := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: accessmodes,
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceName(v1.ResourceStorage): capacity,
				},
			},
			Selector: selector,
		},
		Status: v1.PersistentVolumeClaimStatus{},
	}
	return claim
}
