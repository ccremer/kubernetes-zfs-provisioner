package provisioner

import (
	"context"
	"fmt"
	"slices"
	"strconv"

	"github.com/ccremer/kubernetes-zfs-provisioner/pkg/zfs"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/sig-storage-lib-external-provisioner/v11/controller"
)

// Provision creates a PersistentVolume, sets quota and shares it via NFS.
func (p *ZFSProvisioner) Provision(ctx context.Context, options controller.ProvisionOptions) (*v1.PersistentVolume, controller.ProvisioningState, error) {
	parameters, err := NewStorageClassParameters(options.StorageClass.Parameters)
	if err != nil {
		return nil, controller.ProvisioningNoChange, fmt.Errorf("failed to parse StorageClass parameters: %w", err)
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

	dataset, err := p.zfs.CreateDataset(datasetPath, parameters.Hostname, properties)
	if err != nil {
		return nil, controller.ProvisioningFinished, fmt.Errorf("creating ZFS dataset failed: %w", err)
	}
	if err := p.zfs.SetPermissions(dataset); err != nil {
		return nil, controller.ProvisioningFinished, err
	}
	p.log.Info("dataset created", "dataset", dataset.Name)

	// Copy the annotations from the claim and overwrite with ours
	if options.PVC.Annotations == nil {
		options.PVC.Annotations = make(map[string]string)
	}
	annotations := options.PVC.Annotations
	annotations[DatasetPathAnnotation] = dataset.Name
	annotations[ZFSHostAnnotation] = parameters.Hostname

	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:        options.PVName,
			Labels:      options.PVC.Labels,
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
		},
	}
	return pv, controller.ProvisioningFinished, nil
}

func canUseHostPath(parameters *ZFSStorageClassParameters, options controller.ProvisionOptions) bool {
	switch parameters.Type {
	case Nfs:
		return false
	case HostPath:
		return true
	case Auto:
		if !slices.Contains(options.PVC.Spec.AccessModes, v1.ReadOnlyMany) && !slices.Contains(options.PVC.Spec.AccessModes, v1.ReadWriteMany) {
			return true
		}
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
