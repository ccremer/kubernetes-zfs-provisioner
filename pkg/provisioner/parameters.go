package provisioner

import (
	"fmt"
	"strings"
)

const (
	parentDatasetParameter   = "parentDataset"
	sharePropertiesParameter = "shareProperties"
	hostnameParameter        = "hostname"
	typeParameter            = "type"
	nodeNameParameter        = "node"
)

// StorageClass Parameters are expected in the following schema:
/*
parameters:
  parentDataset: tank/volumes
  hostname: my-zfs-host.localdomain
  type: nfs|hostpath
  shareProperties: rw=10.0.0.0/8,no_root_squash
  node: my-zfs-host
*/

type (
	// ZFSStorageClassParameters represents the parameters on the `StorageClass`
	// object. It is used to ease access and validate those parameters at run time.
	ZFSStorageClassParameters struct {
		// ParentDataset of the zpool. Needs to be existing on the target ZFS host.
		ParentDataset string
		// Hostname of the target ZFS host. Will be used to connect over SSH.
		Hostname string
		NFS      *NFSParameters
		HostPath *HostPathParameters
	}
	NFSParameters struct {
		// ShareProperties specifies additional properties to pass to 'zfs create sharenfs=%s'.
		ShareProperties string
	}
	HostPathParameters struct {
		// NodeName overrides the hostname if the Kubernetes node name is different than the ZFS target host. Used for Affinity
		NodeName string
	}
)

// NewStorageClassParameters takes a storage class parameters, validates it for invalid configuration and returns a
// ZFSStorageClassParameters on success.
func NewStorageClassParameters(parameters map[string]string) (*ZFSStorageClassParameters, error) {
	for _, parameter := range []string{parentDatasetParameter, hostnameParameter, typeParameter} {
		value := parameters[parameter]
		if value == "" {
			return nil, fmt.Errorf("undefined required parameter: %s", parameter)
		}
	}
	parentDataset := parameters[parentDatasetParameter]
	if strings.HasPrefix(parentDataset, "/") || strings.HasSuffix(parentDataset, "/") {
		return nil, fmt.Errorf("%s must not begin or end with '/': %s", parentDatasetParameter, parentDataset)
	}
	p := &ZFSStorageClassParameters{
		ParentDataset: parentDataset,
		Hostname:      parameters[hostnameParameter],
	}
	typeParam := parameters[typeParameter]
	switch typeParam {
	case "hostpath", "hostPath", "HostPath", "Hostpath", "HOSTPATH":
		p.HostPath = &HostPathParameters{NodeName: parameters[nodeNameParameter]}
		return p, nil
	case "nfs", "Nfs", "NFS":
		shareProps := parameters[sharePropertiesParameter]
		if shareProps == "" {
			shareProps = "on"
		}
		p.NFS = &NFSParameters{ShareProperties: shareProps}
		return p, nil
	default:
		return nil, fmt.Errorf("invalid '%s' parameter value: %s", typeParameter, typeParam)
	}
}
