package provisioner

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	ParentDatasetParameter    = "parentDataset"
	SharePropertiesParameter  = "shareProperties"
	HostnameParameter         = "hostname"
	TypeParameter             = "type"
	NodeNameParameter         = "node"
	ReserveSpaceParameter     = "reserveSpace"
	DestroySnapshotsParameter = "destroySnapshots"
)

var (
	// ZFS dataset components: letters, digits, and a small set of punctuation.
	// Rejects "..", shell metacharacters and leading/trailing slashes (checked separately).
	datasetNameRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.:-]*(/[A-Za-z0-9_.:-]+)*$`)
	hostnameRe    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]*$`)
)

// StorageClass Parameters are expected in the following schema:
/*
parameters:
  parentDataset: tank/volumes
  hostname: my-zfs-host.localdomain
  type: nfs|hostPath|auto
  shareProperties: rw=10.0.0.0/8,no_root_squash
  node: my-zfs-host
  reserveSpace: true|false
*/

type ProvisioningType string

const (
	Nfs      ProvisioningType = "nfs"
	HostPath ProvisioningType = "hostPath"
	Auto     ProvisioningType = "auto"
)

type (
	// ZFSStorageClassParameters represents the parameters on the `StorageClass`
	// object. It is used to ease access and validate those parameters at run time.
	ZFSStorageClassParameters struct {
		// ParentDataset of the zpool. Needs to be existing on the target ZFS host.
		ParentDataset string
		// Hostname of the target ZFS host. Will be used to connect over SSH.
		Hostname string
		Type     ProvisioningType
		// NFSShareProperties specifies additional properties to pass to 'zfs create sharenfs=%s'.
		NFSShareProperties string
		// HostPathNodeName overrides the hostname if the Kubernetes node name is different than the ZFS target host. Used for Affinity
		HostPathNodeName string
		ReserveSpace     bool
		// DestroySnapshots, when true (the default, matching historic destroy -r),
		// allows Delete to recursively remove user snapshots. Set false to refuse
		// deletion while snapshots exist.
		DestroySnapshots bool
	}
)

// NewStorageClassParameters takes a storage class parameters, validates it for invalid configuration and returns a
// ZFSStorageClassParameters on success.
func NewStorageClassParameters(parameters map[string]string) (*ZFSStorageClassParameters, error) {
	for _, parameter := range []string{ParentDatasetParameter, HostnameParameter, TypeParameter} {
		value := parameters[parameter]
		if value == "" {
			return nil, fmt.Errorf("undefined required parameter: %s", parameter)
		}
	}
	parentDataset := parameters[ParentDatasetParameter]
	if strings.HasPrefix(parentDataset, "/") || strings.HasSuffix(parentDataset, "/") {
		return nil, fmt.Errorf("%s must not begin or end with '/': %s", ParentDatasetParameter, parentDataset)
	}
	if strings.Contains(parentDataset, "..") || !datasetNameRe.MatchString(parentDataset) {
		return nil, fmt.Errorf("%s contains invalid characters: %s", ParentDatasetParameter, parentDataset)
	}
	hostname := parameters[HostnameParameter]
	if !hostnameRe.MatchString(hostname) {
		return nil, fmt.Errorf("%s contains invalid characters: %s", HostnameParameter, hostname)
	}

	reserveSpaceValue, reserveSpaceValuePresent := parameters[ReserveSpaceParameter]
	var reserveSpace bool
	if !reserveSpaceValuePresent || strings.EqualFold(reserveSpaceValue, "true") {
		reserveSpace = true
	} else if strings.EqualFold(reserveSpaceValue, "false") {
		reserveSpace = false
	} else {
		return nil, fmt.Errorf("invalid '%s' parameter value: %s", ReserveSpaceParameter, parameters[ReserveSpaceParameter])
	}

	destroySnaps, err := parseBoolParam(parameters, DestroySnapshotsParameter, true)
	if err != nil {
		return nil, err
	}

	p := &ZFSStorageClassParameters{
		ParentDataset:    parentDataset,
		Hostname:         hostname,
		ReserveSpace:     reserveSpace,
		DestroySnapshots: destroySnaps,
	}
	typeParam := parameters[TypeParameter]
	switch typeParam {
	case "hostpath", "hostPath", "HostPath", "Hostpath", "HOSTPATH":
		p.Type = HostPath
	case "nfs", "Nfs", "NFS":
		p.Type = Nfs
	case "auto", "Auto", "AUTO":
		p.Type = Auto
	default:
		return nil, fmt.Errorf("invalid '%s' parameter value: %s", TypeParameter, typeParam)
	}

	if p.Type == HostPath || p.Type == Auto {
		p.HostPathNodeName = parameters[NodeNameParameter]
	}

	if p.Type == Nfs || p.Type == Auto {
		shareProps := parameters[SharePropertiesParameter]
		if shareProps == "" {
			shareProps = "on"
		}
		p.NFSShareProperties = shareProps
	}

	return p, nil
}

func parseBoolParam(parameters map[string]string, key string, defaultVal bool) (bool, error) {
	v, ok := parameters[key]
	if !ok || v == "" {
		return defaultVal, nil
	}
	if strings.EqualFold(v, "true") {
		return true, nil
	}
	if strings.EqualFold(v, "false") {
		return false, nil
	}
	return false, fmt.Errorf("invalid '%s' parameter value: %s", key, v)
}
