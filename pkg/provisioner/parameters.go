package provisioner

import (
	"fmt"
	"net"
)

const (
	scParametersParentDataset = "parentDataset"
	scParametersShareSubnet   = "shareSubnet"
	scParametersShareOptions  = "shareOptions"
	scParametersHostname      = "hostname"
)

// ZFSStorageClassParameters represents the parameters on the `StorageClas`
// object. It is used to ease access and validate those parameters at run time.
type ZFSStorageClassParameters struct {
	ParentDataset string
	ShareSubnet   string
	ShareOptions  string
	Hostname      string
}

// NewStorageClassParameters takes a storage classes parameters as string map,
// validates it for invalid configuration and returns a
// ZFSStorageClassParameters on success.
func NewStorageClassParameters(parameters map[string]string) (*ZFSStorageClassParameters, error) {
	parentDataset, ok := parameters[scParametersParentDataset]
	if !ok {
		return nil, fmt.Errorf("no parentDataset defined")
	}

	shareSubnet, ok := parameters[scParametersShareSubnet]
	if !ok {
		return nil, fmt.Errorf("no shareSubnet defined")
	}
	if _, _, err := net.ParseCIDR(shareSubnet); err != nil {
		return nil, fmt.Errorf("shareSubnet is invalid: %v", parameters[scParametersShareSubnet])
	}

	shareOptions := parameters[scParametersShareOptions]

	hostname, ok := parameters[scParametersHostname]
	if !ok {
		return nil, fmt.Errorf("no hostname defined")
	}

	p := &ZFSStorageClassParameters{
		ParentDataset: parentDataset,
		ShareSubnet:   shareSubnet,
		ShareOptions:  shareOptions,
		Hostname:      hostname,
	}

	return p, nil
}
