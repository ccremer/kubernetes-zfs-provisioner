package provisioner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	core "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ccremer/kubernetes-zfs-provisioner/pkg/zfs"
)

func TestDelete_GivenVolume_WhenAnnotationCorrect_ThenDeleteZfsDataset(t *testing.T) {
	expectedDataset := "test/volumes/pv-testcreate"
	expectedHost := "host"
	dataset := &zfs.Dataset{
		Name:     expectedDataset,
		Hostname: expectedHost,
	}
	stub := new(zfsStub)
	stub.On("DestroyDataset", dataset, zfs.DestroyRecursively).
		Return(nil)
	p, _ := NewZFSProvisionerStub(stub)
	pv := core.PersistentVolume{ObjectMeta: v1.ObjectMeta{Annotations: map[string]string{
		DatasetPathAnnotation: expectedDataset,
		ZFSHostAnnotation:     expectedHost,
	}}}
	result := p.Delete(context.Background(), &pv)
	require.NoError(t, result)
	stub.AssertExpectations(t)
}

func TestDelete_GivenVolume_WhenAnnotationMissing_ThenThrowError(t *testing.T) {
	stub := new(zfsStub)
	p, _ := NewZFSProvisionerStub(stub)
	pv := core.PersistentVolume{}
	err := p.Delete(context.Background(), &pv)
	require.Error(t, err)
	stub.AssertExpectations(t)
	assert.Contains(t, err.Error(), "annotation")
}

func TestDelete_RefusesWhenSnapshotsExist(t *testing.T) {
	dataset := &zfs.Dataset{Name: "tank/volumes/pv-x", Hostname: "nas-1"}
	stub := new(zfsStub)
	stub.On("ListSnapshots", dataset).Return([]string{"tank/volumes/pv-x@keep"}, nil)
	p, _ := NewZFSProvisionerStub(stub)
	err := p.Delete(context.Background(), &core.PersistentVolume{ObjectMeta: v1.ObjectMeta{Annotations: map[string]string{
		DatasetPathAnnotation:      "tank/volumes/pv-x",
		ZFSHostAnnotation:          "nas-1",
		DestroySnapshotsAnnotation: "false",
	}}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "snapshot")
	stub.AssertNotCalled(t, "DestroyDataset", mock.Anything, mock.Anything)
}
