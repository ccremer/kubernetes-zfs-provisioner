package provisioner

import (
	"github.com/ccremer/kubernetes-zfs-provisioner/pkg/zfs"
	gozfs "github.com/mistifyio/go-zfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	core "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestDelete_GivenVolume_WhenAnnotationCorrect_ThenDeleteZfsDataset(t *testing.T) {
	expectedDataset := "test/volumes/pv-testcreate"
	expectedHost := "host"
	dataset := &zfs.Dataset{
		Name:     expectedDataset,
		Hostname: expectedHost,
	}
	stub := new(zfsStub)
	stub.On("DestroyDataset", dataset, zfs.DestroyFlag(gozfs.DestroyRecursive)).
		Return(nil)
	p, _ := NewZFSProvisionerStub(stub)
	pv := core.PersistentVolume{ObjectMeta: v1.ObjectMeta{Annotations: map[string]string{
		DatasetPathAnnotation: expectedDataset,
		ZFSHostAnnotation:     expectedHost,
	}}}
	result := p.Delete(&pv)
	require.NoError(t, result)
	stub.AssertExpectations(t)
}

func TestDelete_GivenVolume_WhenAnnotationMissing_ThenThrowError(t *testing.T) {
	stub := new(zfsStub)
	p, _ := NewZFSProvisionerStub(stub)
	pv := core.PersistentVolume{}
	err := p.Delete(&pv)
	require.Error(t, err)
	stub.AssertExpectations(t)
	assert.Contains(t, err.Error(), "annotation")
}
