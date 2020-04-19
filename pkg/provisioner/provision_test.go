package provisioner

import (
	"github.com/ccremer/kubernetes-zfs-provisioner/pkg/zfs"
	"github.com/stretchr/testify/require"
	storagev1 "k8s.io/api/storage/v1"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/sig-storage-lib-external-provisioner/controller"
)

func TestProvisionNfs(t *testing.T) {

	expectedShareProperties := "rw=@10.0.0.0/8"
	expectedHost := "host"
	expectedDatasetName := "test/volumes/pv-testcreate"
	expectedDataset := &zfs.Dataset{
		Name:       expectedDatasetName,
		Hostname:   expectedHost,
		Mountpoint: "/" + expectedDatasetName,
	}
	stub := new(zfsStub)
	stub.On("CreateDataset", expectedDatasetName, map[string]string{
		RefQuotaProperty:       "1000000000",
		RefReservationProperty: "1000000000",
		"sharenfs":             "rw=@10.0.0.0/8",
		ManagedByProperty:      "test",
		ReclaimPolicyProperty:  string(v1.PersistentVolumeReclaimDelete),
	}).Return(expectedDataset, nil)
	stub.On("SetPermissions", expectedDataset).Return(nil)

	p, _ := NewZFSProvisionerStub(stub)
	options := controller.ProvisionOptions{
		PVName: "pv-testcreate",
		PVC:    newClaim(resource.MustParse("1G"), []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce, v1.ReadOnlyMany}),
		StorageClass: &storagev1.StorageClass{
			Parameters: map[string]string{
				ParentDatasetParameter:   "test/volumes",
				HostnameParameter:        expectedHost,
				TypeParameter:            "nfs",
				SharePropertiesParameter: expectedShareProperties,
			},
		},
	}

	pv, err := p.Provision(options)
	require.NoError(t, err)
	assertBasics(t, stub, pv, expectedDatasetName, expectedHost)

	assert.Equal(t, v1.PersistentVolumeReclaimDelete, pv.Spec.PersistentVolumeReclaimPolicy)

	require.NotNil(t, pv.Spec.NFS)
	require.Nil(t, pv.Spec.HostPath)
	require.Nil(t, pv.Spec.NodeAffinity)
	assert.Equal(t, "/"+expectedDatasetName, pv.Spec.NFS.Path)
	assert.Equal(t, expectedHost, pv.Spec.NFS.Server)
}

func assertBasics(t *testing.T, stub *zfsStub, pv *v1.PersistentVolume, expectedDataset string, expectedHost string) {
	stub.AssertExpectations(t)

	assert.Contains(t, pv.Spec.AccessModes, v1.ReadWriteOnce)
	assert.Contains(t, pv.Spec.AccessModes, v1.ReadOnlyMany)
	assert.Contains(t, pv.Spec.AccessModes, v1.ReadWriteMany)

	assert.Contains(t, pv.Annotations, "my/annotation")
	assert.Equal(t, expectedDataset, pv.Annotations[DatasetPathAnnotation])
	assert.Equal(t, expectedHost, pv.Annotations[ZFSHostAnnotation])
}

func TestProvisionHostPath(t *testing.T) {

	expectedDatasetName := "test/volumes/pv-testcreate"
	expectedDataset := &zfs.Dataset{
		Name:       expectedDatasetName,
		Mountpoint: "/" + expectedDatasetName,
	}
	expectedHost := "host"
	policy := v1.PersistentVolumeReclaimRetain
	stub := new(zfsStub)
	stub.On("CreateDataset", expectedDatasetName, map[string]string{
		RefQuotaProperty:       "1000000000",
		RefReservationProperty: "1000000000",
		ManagedByProperty:      "test",
		ReclaimPolicyProperty:  string(policy),
	}).Return(expectedDataset, nil)
	stub.On("SetPermissions", expectedDataset).Return(nil)

	p, _ := NewZFSProvisionerStub(stub)
	options := controller.ProvisionOptions{
		PVName: "pv-testcreate",
		PVC:    newClaim(resource.MustParse("1G"), []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce, v1.ReadOnlyMany}),
		StorageClass: &storagev1.StorageClass{
			Parameters: map[string]string{
				ParentDatasetParameter: "test/volumes",
				HostnameParameter:      expectedHost,
				TypeParameter:          "hostpath",
				NodeNameParameter:      "node",
			},
			ReclaimPolicy: &policy,
		},
	}

	pv, err := p.Provision(options)
	require.NoError(t, err)
	assertBasics(t, stub, pv, expectedDatasetName, expectedHost)

	assert.Equal(t, policy, pv.Spec.PersistentVolumeReclaimPolicy)

	hostPathType := v1.HostPathDirectory
	require.NotNil(t, pv.Spec.HostPath)
	require.Nil(t, pv.Spec.NFS)
	assert.Equal(t, "/"+expectedDatasetName, pv.Spec.HostPath.Path)
	assert.Equal(t, &hostPathType, pv.Spec.HostPath.Type)
	assert.Contains(t, pv.Spec.NodeAffinity.Required.NodeSelectorTerms[0].MatchExpressions[0].Values, "node")
}

func newClaim(capacity resource.Quantity, accessmodes []v1.PersistentVolumeAccessMode) *v1.PersistentVolumeClaim {
	storageClassName := "zfs"
	claim := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"my/annotation": "value",
			},
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: accessmodes,
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: capacity,
				},
			},
			StorageClassName: &storageClassName,
		},
		Status: v1.PersistentVolumeClaimStatus{},
	}
	return claim
}
