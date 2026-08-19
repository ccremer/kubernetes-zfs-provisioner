package provisioner

import (
	"context"
	"testing"

	"github.com/ccremer/kubernetes-zfs-provisioner/pkg/zfs"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestExpandGrowsQuotaAndPV(t *testing.T) {
	allow := true
	sc := &storagev1.StorageClass{
		ObjectMeta:           metav1.ObjectMeta{Name: "zfs"},
		Provisioner:          "test",
		AllowVolumeExpansion: &allow,
	}
	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pv-x",
			Annotations: map[string]string{
				DatasetPathAnnotation:  "tank/volumes/pv-x",
				ZFSHostAnnotation:      "nas-1",
				ReserveSpaceAnnotation: "true",
			},
		},
		Spec: v1.PersistentVolumeSpec{
			StorageClassName: "zfs",
			Capacity:         v1.ResourceList{v1.ResourceStorage: resource.MustParse("1Gi")},
		},
	}
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "claim", Namespace: "ns"},
		Spec: v1.PersistentVolumeClaimSpec{
			VolumeName:       "pv-x",
			StorageClassName: strPtr("zfs"),
			Resources: v1.VolumeResourceRequirements{
				Requests: v1.ResourceList{v1.ResourceStorage: resource.MustParse("2Gi")},
			},
		},
		Status: v1.PersistentVolumeClaimStatus{Phase: v1.ClaimBound},
	}
	kube := fake.NewSimpleClientset(sc, pv, pvc)
	ds := &zfs.Dataset{Name: "tank/volumes/pv-x", Hostname: "nas-1"}
	stub := new(zfsStub)
	stub.On("SetProperty", ds, RefQuotaProperty, "2147483648").Return(nil)
	stub.On("SetProperty", ds, RefReservationProperty, "2147483648").Return(nil)
	p, _ := NewZFSProvisionerStub(stub)

	require.NoError(t, p.expandOnce(context.Background(), kube))

	got, err := kube.CoreV1().PersistentVolumes().Get(context.Background(), "pv-x", metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, int64(2147483648), got.Spec.Capacity.Storage().Value())
	stub.AssertExpectations(t)
}

func TestExpandIgnoresShrink(t *testing.T) {
	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pv-x",
			Annotations: map[string]string{
				DatasetPathAnnotation: "tank/volumes/pv-x",
				ZFSHostAnnotation:     "nas-1",
			},
		},
		Spec: v1.PersistentVolumeSpec{
			Capacity: v1.ResourceList{v1.ResourceStorage: resource.MustParse("2Gi")},
		},
	}
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "claim", Namespace: "ns"},
		Spec: v1.PersistentVolumeClaimSpec{
			VolumeName: "pv-x",
			Resources: v1.VolumeResourceRequirements{
				Requests: v1.ResourceList{v1.ResourceStorage: resource.MustParse("1Gi")},
			},
		},
		Status: v1.PersistentVolumeClaimStatus{Phase: v1.ClaimBound},
	}
	kube := fake.NewSimpleClientset(pv, pvc)
	stub := new(zfsStub)
	p, _ := NewZFSProvisionerStub(stub)
	require.NoError(t, p.expandOnce(context.Background(), kube))
	stub.AssertExpectations(t)
}

func strPtr(s string) *string { return &s }
