package provisioner

import "github.com/kubernetes-incubator/external-storage/lib/controller"

const (
	annCreatedBy = "kubernetes.io/createdby"
	createdBy    = "zfs-provisioner"
)

// ZFSProvisioner implements the Provisioner interface to create and export ZFS volumes
type ZFSProvisioner struct {
	zpool         string // The Zpool in which to create volume
	mountPrefix   string // The path where the zpool is mounted, e.g. /Volumes/ under MacOS
	parentDataset string // The parent dataset under which tho create volumes

	shareOptions   string // Additional nfs export options, comma-separated
	shareSubnet    string // The subnet to which the volumes will be exported
	serverHostname string // The hostname that should be returned as NFS Server
}

// NewZFSProvisioner returns a new ZFSProvisioner
func NewZFSProvisioner(zpool string, mountPrefix string, parentDataset string, shareOptions string, shareSubnet string, serverHostname string) controller.Provisioner {
	// Prepend a comma if additional options are given
	if shareOptions != "" {
		shareOptions = "," + shareOptions
	}

	return ZFSProvisioner{
		zpool:         zpool,
		mountPrefix:   mountPrefix,
		parentDataset: parentDataset,

		shareOptions:   shareOptions,
		shareSubnet:    shareSubnet,
		serverHostname: serverHostname,
	}
}
