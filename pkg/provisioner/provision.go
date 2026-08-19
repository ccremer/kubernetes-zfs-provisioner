package provisioner

import (
	"context"
	"fmt"
	"slices"
	"strconv"

	"github.com/ccremer/kubernetes-zfs-provisioner/pkg/zfs"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/sig-storage-lib-external-provisioner/v13/controller"
)

const (
	DestroySnapshotsAnnotation = "zfs.pv.kubernetes.io/destroy-snapshots"
	ReserveSpaceAnnotation     = "zfs.pv.kubernetes.io/reserve-space"
)

// Provision creates a PersistentVolume, sets quota and shares it via NFS.
func (p *ZFSProvisioner) Provision(ctx context.Context, options controller.ProvisionOptions) (*v1.PersistentVolume, controller.ProvisioningState, error) {
	parameters, err := NewStorageClassParameters(options.StorageClass.Parameters)
	if err != nil {
		// Invalid spec will not start working on retry.
		return nil, controller.ProvisioningFinished, fmt.Errorf("failed to parse StorageClass parameters: %w", err)
	}

	datasetPath := fmt.Sprintf("%s/%s", parameters.ParentDataset, options.PVName)
	properties := make(map[string]string)

	useHostPath := canUseHostPath(parameters, options)
	if !useHostPath {
		properties[ShareNfsProperty] = parameters.NFSShareProperties
	}

	var reclaimPolicy v1.PersistentVolumeReclaimPolicy
	if options.StorageClass.ReclaimPolicy == nil {
		// Default is delete, see https://kubernetes.io/docs/concepts/storage/storage-classes/#reclaim-policy
		reclaimPolicy = v1.PersistentVolumeReclaimDelete
	} else if *options.StorageClass.ReclaimPolicy == v1.PersistentVolumeReclaimRecycle {
		return nil, controller.ProvisioningFinished, fmt.Errorf("unsupported reclaim policy of this provisioner: %s", v1.PersistentVolumeReclaimRecycle)
	} else {
		reclaimPolicy = *options.StorageClass.ReclaimPolicy
	}

	storageRequest := options.PVC.Spec.Resources.Requests[v1.ResourceStorage]
	storageRequestBytes := strconv.FormatInt(storageRequest.Value(), 10)
	properties[RefQuotaProperty] = storageRequestBytes
	properties[ManagedByProperty] = p.InstanceName
	properties[ReclaimPolicyProperty] = string(reclaimPolicy)

	if parameters.ReserveSpace {
		properties[RefReservationProperty] = storageRequestBytes
	}

	p.log.Info("provisioning volume",
		"pvc", options.PVC.Namespace+"/"+options.PVC.Name,
		"pv", options.PVName,
		"dataset", datasetPath,
		"host", parameters.Hostname,
		"hostPath", useHostPath,
	)

	if err := p.checkCapacity(parameters, storageRequest.Value()); err != nil {
		return nil, controller.ProvisioningFinished, err
	}

	dataset, err := p.zfs.CreateDataset(datasetPath, parameters.Hostname, properties)
	if err != nil {
		return nil, retryState(err), fmt.Errorf("creating ZFS dataset failed: %w", err)
	}
	if err := p.zfs.SetPermissions(dataset); err != nil {
		// Leave the dataset in place: a retry (InBackground) will adopt it
		// via idempotent create and re-run chmod.
		return nil, retryState(err), err
	}
	p.log.Info("dataset created",
		"dataset", dataset.Name,
		"host", parameters.Hostname,
		"pvc", options.PVC.Namespace+"/"+options.PVC.Name,
	)

	annotations := map[string]string{
		DatasetPathAnnotation:      dataset.Name,
		ZFSHostAnnotation:          parameters.Hostname,
		DestroySnapshotsAnnotation: strconv.FormatBool(parameters.DestroySnapshots),
		ReserveSpaceAnnotation:     strconv.FormatBool(parameters.ReserveSpace),
	}

	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:        options.PVName,
			Annotations: annotations,
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: reclaimPolicy,
			AccessModes:                   createAccessModes(options, useHostPath),
			Capacity: v1.ResourceList{
				v1.ResourceStorage: options.PVC.Spec.Resources.Requests[v1.ResourceStorage],
			},
			PersistentVolumeSource: createVolumeSource(parameters, dataset, useHostPath),
			NodeAffinity:           createNodeAffinity(parameters, useHostPath),
			MountOptions:           options.StorageClass.MountOptions,
			StorageClassName:       options.StorageClass.Name,
		},
	}
	return pv, controller.ProvisioningFinished, nil
}

func (p *ZFSProvisioner) checkCapacity(parameters *ZFSStorageClassParameters, needBytes int64) error {
	if needBytes <= 0 {
		return nil
	}
	availStr, err := p.zfs.GetProperty(parameters.ParentDataset, parameters.Hostname, "available")
	if err != nil {
		// Do not fail provision if we cannot read capacity (permissions, old mock).
		p.log.Info("skipping capacity pre-check", "host", parameters.Hostname, "err", err.Error())
		return nil
	}
	avail, ok := zfs.ParseAvailableBytes(availStr)
	if !ok {
		return nil
	}
	if avail < needBytes {
		return fmt.Errorf("insufficient capacity on %s:%s: need %d bytes, available %d",
			parameters.Hostname, parameters.ParentDataset, needBytes, avail)
	}
	return nil
}

// retryState returns InBackground so the controller retries transient backend
// failures. ProvisioningNoChange on a *first* call is treated as Finished by
// the library, so it must not be used for "please try again".
func retryState(err error) controller.ProvisioningState {
	if zfs.IsTransient(err) {
		return controller.ProvisioningInBackground
	}
	return controller.ProvisioningFinished
}

func canUseHostPath(parameters *ZFSStorageClassParameters, options controller.ProvisionOptions) bool {
	switch parameters.Type {
	case Nfs:
		return false
	case HostPath:
		return true
	case Auto:
		if options.SelectedNodeName != "" {
			return selectedNodeIsZFSHost(parameters, options.SelectedNodeName)
		}
		// Immediate binding: fall back to access modes (RWO/RWOP → HostPath).
		if !slices.Contains(options.PVC.Spec.AccessModes, v1.ReadOnlyMany) && !slices.Contains(options.PVC.Spec.AccessModes, v1.ReadWriteMany) {
			return true
		}
	}
	return false
}

func selectedNodeIsZFSHost(parameters *ZFSStorageClassParameters, selected string) bool {
	if selected == parameters.Hostname {
		return true
	}
	if parameters.HostPathNodeName != "" && selected == parameters.HostPathNodeName {
		return true
	}
	return false
}

func createAccessModes(options controller.ProvisionOptions, useHostPath bool) []v1.PersistentVolumeAccessMode {
	if slices.Contains(options.PVC.Spec.AccessModes, v1.ReadWriteOncePod) {
		return []v1.PersistentVolumeAccessMode{v1.ReadWriteOncePod}
	}
	accessModes := []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce}
	if !useHostPath {
		accessModes = append(accessModes, v1.ReadOnlyMany, v1.ReadWriteMany)
	}
	return accessModes
}

func createVolumeSource(parameters *ZFSStorageClassParameters, dataset *zfs.Dataset, useHostPath bool) v1.PersistentVolumeSource {
	if useHostPath {
		hostPathType := v1.HostPathDirectory
		return v1.PersistentVolumeSource{
			HostPath: &v1.HostPathVolumeSource{
				Path: dataset.Mountpoint,
				Type: &hostPathType,
			},
		}
	}

	return v1.PersistentVolumeSource{
		NFS: &v1.NFSVolumeSource{
			Server:   parameters.Hostname,
			Path:     dataset.Mountpoint,
			ReadOnly: false,
		},
	}
}

func createNodeAffinity(parameters *ZFSStorageClassParameters, useHostPath bool) *v1.VolumeNodeAffinity {
	if !useHostPath {
		return nil
	}

	node := parameters.HostPathNodeName
	if node == "" {
		node = parameters.Hostname
	}
	return &v1.VolumeNodeAffinity{Required: &v1.NodeSelector{NodeSelectorTerms: []v1.NodeSelectorTerm{
		{
			MatchExpressions: []v1.NodeSelectorRequirement{
				{
					Values:   []string{node},
					Operator: v1.NodeSelectorOpIn,
					Key:      v1.LabelHostname,
				},
			},
		},
	}}}
}
