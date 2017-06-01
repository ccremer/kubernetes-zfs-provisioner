package provisioner

import (
	"os"
	"testing"

	"github.com/kubernetes-incubator/external-storage/lib/controller"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/pkg/api/v1"
)

func TestDelete(t *testing.T) {
	p := NewZFSProvisioner("test", "", "volumes", "", "127.0.0.1", "", "Retain")
	options := controller.VolumeOptions{
		PersistentVolumeReclaimPolicy: v1.PersistentVolumeReclaimDelete,
		PVName: "pv-testdelete",
		PVC:    newClaim(resource.MustParse("1G"), []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce, v1.ReadOnlyMany}, nil),
	}
	pv, _ := p.Provision(options)

	err := p.Delete(pv)
	assert.NoError(t, err, "Delete should not return an error")

	_, err = os.Stat(os.Getenv("GOPATH") + "/src/git.gentics.com/psc/kubernetes-zfs-provisioner/" + pv.Spec.PersistentVolumeSource.NFS.Path)
	assert.Error(t, err, "The volume should not exist on disk")
}
