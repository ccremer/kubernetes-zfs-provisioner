// +build integration

package provisioner

import (
	"go.uber.org/zap"
	storagev1 "k8s.io/api/storage/v1"
	"os"
	"os/user"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/sig-storage-lib-external-provisioner/controller"
)

func TestProvision(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	p, _ := NewZFSProvisioner(logger)
	options := controller.ProvisionOptions{
		PVName:                        "pv-testcreate",
		PVC:                           newClaim(resource.MustParse("1G"), []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce, v1.ReadOnlyMany}, nil),
		StorageClass:                  &storagev1.StorageClass{
			Parameters: map[string]string{
				"parentDataset": "test/volumes",
				"shareSubnet": "10.0.0.0/8",
				"hostname": "test",
			},
		},
	}

	pv, err := p.Provision(options)

	assert.NoError(t, err, "Provision should not return an error")
	// u, _ := user.Current()
	// name := u.Username
	// log.Fatalf("I am %s: %s", name, err.Error())
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

func TestBla(t *testing.T) {
	u, _ := user.Current()
	t.Logf("I am %s", u.Uid)
}
