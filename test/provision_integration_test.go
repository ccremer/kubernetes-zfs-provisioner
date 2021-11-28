// +build integration

package test

import (
	"bufio"
	"context"
	"flag"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/sig-storage-lib-external-provisioner/v8/controller"

	"github.com/ccremer/kubernetes-zfs-provisioner/pkg/provisioner"
	"github.com/ccremer/kubernetes-zfs-provisioner/pkg/zfs"
)

var (
	parentDataset = flag.String("parentDataset", "", "parent dataset")
)

type ProvisionTestSuit struct {
	suite.Suite
	p       *provisioner.ZFSProvisioner
	dataset string
}

func TestProvisionSuite(t *testing.T) {
	s := ProvisionTestSuit{
		dataset: "pv-test-" + strconv.Itoa(rand.Int()),
	}
	suite.Run(t, &s)
}

func (suite *ProvisionTestSuit) SetupSuite() {
	path := os.Getenv("PATH")
	pwd, _ := os.Getwd()
	err := os.Setenv("PATH", pwd+":"+path)
	require.NoError(suite.T(), err)
	prov, err := provisioner.NewZFSProvisioner("pv.kubernetes.io/zfs")
	require.NoError(suite.T(), err)
	suite.p = prov
}

func (suite *ProvisionTestSuit) TearDownSuite() {
	err := zfs.NewInterface().DestroyDataset(&zfs.Dataset{
		Name:     *parentDataset + "/" + suite.dataset,
		Hostname: "host",
	}, zfs.DestroyRecursively)
	require.NoError(suite.T(), err)
}

func (suite *ProvisionTestSuit) TestProvisionDataset() {
	t := suite.T()
	fullDataset := "/" + *parentDataset + "/" + suite.dataset
	policy := v1.PersistentVolumeReclaimRetain
	options := controller.ProvisionOptions{
		PVName: suite.dataset,
		PVC:    newClaim(resource.MustParse("10M"), []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce, v1.ReadOnlyMany}),
		StorageClass: &storagev1.StorageClass{
			Parameters: map[string]string{
				provisioner.ParentDatasetParameter:   *parentDataset,
				provisioner.HostnameParameter:        "localhost",
				provisioner.TypeParameter:            "nfs",
				provisioner.SharePropertiesParameter: "rw,no_root_squash",
			},
			ReclaimPolicy: &policy,
		},
	}

	_, _, err := suite.p.Provision(context.Background(), options)
	assert.NoError(t, err)
	require.DirExists(t, fullDataset)
	assertNfsExport(t, fullDataset)
}

func assertNfsExport(t *testing.T, fullDataset string) {
	file, err := os.Open("/var/lib/nfs/etab")
	require.NoError(t, err)
	defer file.Close()

	scanner := bufio.NewScanner(file)
	require.NoError(t, err)
	found := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, fullDataset) {
			found = true
			assert.Contains(t, line, "rw")
			assert.Contains(t, line, "no_root_squash")
		}
	}
	assert.True(t, found)
}

func newClaim(capacity resource.Quantity, accessmodes []v1.PersistentVolumeAccessMode) *v1.PersistentVolumeClaim {
	storageClassName := "zfs"
	claim := &v1.PersistentVolumeClaim{
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: accessmodes,
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: capacity,
				},
			},
			StorageClassName: &storageClassName,
		},
	}
	return claim
}
