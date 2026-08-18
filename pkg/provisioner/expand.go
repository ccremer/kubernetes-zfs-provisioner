package provisioner

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ccremer/kubernetes-zfs-provisioner/pkg/zfs"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

const expandInterval = 15 * time.Second

// RunExpander periodically finds Bound PVCs whose requested size exceeds the
// backing PV capacity and grows the ZFS refquota (and refreservation when the
// volume was thick-provisioned). Filesystem resize on the node is not required:
// a ZFS dataset quota change is visible immediately over NFS and HostPath.
func RunExpander(ctx context.Context, kube kubernetes.Interface, p *ZFSProvisioner, log klog.Logger) {
	log.Info("starting volume expander")
	ticker := time.NewTicker(expandInterval)
	defer ticker.Stop()
	for {
		if err := p.expandOnce(ctx, kube); err != nil {
			log.Error(err, "expand reconcile failed")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (p *ZFSProvisioner) expandOnce(ctx context.Context, kube kubernetes.Interface) error {
	pvcs, err := kube.CoreV1().PersistentVolumeClaims("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("list pvcs: %w", err)
	}
	for i := range pvcs.Items {
		pvc := &pvcs.Items[i]
		if pvc.Status.Phase != v1.ClaimBound || pvc.Spec.VolumeName == "" {
			continue
		}
		if err := p.expandPVC(ctx, kube, pvc); err != nil {
			p.log.Error(err, "expand pvc failed", "pvc", pvc.Namespace+"/"+pvc.Name)
		}
	}
	return nil
}

func (p *ZFSProvisioner) expandPVC(ctx context.Context, kube kubernetes.Interface, pvc *v1.PersistentVolumeClaim) error {
	pv, err := kube.CoreV1().PersistentVolumes().Get(ctx, pvc.Spec.VolumeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get pv %s: %w", pvc.Spec.VolumeName, err)
	}
	if pv.Annotations[ZFSHostAnnotation] == "" || pv.Annotations[DatasetPathAnnotation] == "" {
		return nil
	}
	if pv.Spec.StorageClassName != "" && pvc.Spec.StorageClassName != nil &&
		pv.Spec.StorageClassName != *pvc.Spec.StorageClassName {
		return nil
	}
	if scName := pv.Spec.StorageClassName; scName != "" {
		sc, err := kube.StorageV1().StorageClasses().Get(ctx, scName, metav1.GetOptions{})
		if err == nil && sc.AllowVolumeExpansion != nil && !*sc.AllowVolumeExpansion {
			return nil
		}
		if err == nil && sc.Provisioner != p.InstanceName {
			return nil
		}
	}

	want, ok := pvc.Spec.Resources.Requests[v1.ResourceStorage]
	if !ok {
		return nil
	}
	have, ok := pv.Spec.Capacity[v1.ResourceStorage]
	if !ok {
		return nil
	}
	switch want.Cmp(have) {
	case 0:
		return nil
	case -1:
		p.log.Info("ignoring shrink request", "pvc", pvc.Namespace+"/"+pvc.Name, "want", want.String(), "have", have.String())
		return nil
	}

	dataset := &zfs.Dataset{
		Name:     pv.Annotations[DatasetPathAnnotation],
		Hostname: pv.Annotations[ZFSHostAnnotation],
	}
	bytes := strconv.FormatInt(want.Value(), 10)
	if err := p.zfs.SetProperty(dataset, RefQuotaProperty, bytes); err != nil {
		return fmt.Errorf("set refquota: %w", err)
	}
	if reserveSpaceEnabled(pv) {
		if err := p.zfs.SetProperty(dataset, RefReservationProperty, bytes); err != nil {
			return fmt.Errorf("set refreservation: %w", err)
		}
	}

	pv.Spec.Capacity[v1.ResourceStorage] = want
	if _, err := kube.CoreV1().PersistentVolumes().Update(ctx, pv, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update pv capacity: %w", err)
	}
	if pvc.Status.Capacity == nil {
		pvc.Status.Capacity = v1.ResourceList{}
	}
	pvc.Status.Capacity[v1.ResourceStorage] = want
	if _, err := kube.CoreV1().PersistentVolumeClaims(pvc.Namespace).UpdateStatus(ctx, pvc, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update pvc status capacity: %w", err)
	}
	p.log.Info("volume expanded",
		"pvc", pvc.Namespace+"/"+pvc.Name,
		"pv", pv.Name,
		"from", have.String(),
		"to", want.String(),
		"dataset", dataset.Name,
	)
	return nil
}

func reserveSpaceEnabled(pv *v1.PersistentVolume) bool {
	v := pv.Annotations[ReserveSpaceAnnotation]
	if v == "" {
		return true
	}
	ok, err := strconv.ParseBool(v)
	if err != nil {
		return true
	}
	return ok
}

// ExpandQuantity is exported for tests.
func ExpandQuantity(s string) resource.Quantity {
	return resource.MustParse(s)
}
