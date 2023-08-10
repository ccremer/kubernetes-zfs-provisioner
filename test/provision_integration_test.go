// +build integration

package test

import (
	"bufio"
	"context"
	"flag"
	gozfs "github.com/mistifyio/go-zfs/v3"
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
)

var (
	parentDataset = flag.String("parentDataset", "", "parent dataset")
)

type ProvisionTestSuit struct {
	suite.Suite
	p               *provisioner.ZFSProvisioner
	datasetPrefix   string
	createdDatasets []string
}

func TestProvisionSuite(t *testing.T) {
	s := ProvisionTestSuit{
		datasetPrefix:   "pv-test-" + strconv.Itoa(rand.Int()),
		createdDatasets: make([]string, 0),
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
	for _, dataset := range suite.createdDatasets {
		err := zfs.NewInterface().DestroyDataset(&zfs.Dataset{
			Name:     *parentDataset + "/" + dataset,
			Hostname: "host",
		}, zfs.DestroyRecursively)
		require.NoError(suite.T(), err)
	}
}

func (suite *ProvisionTestSuit) TestDefaultProvisionDataset() {
	dataset := provisionDataset(suite, "default", map[string]string{
		provisioner.ParentDatasetParameter:   *parentDataset,
		provisioner.HostnameParameter:        "localhost",
		provisioner.TypeParameter:            "nfs",
		provisioner.SharePropertiesParameter: "rw,no_root_squash",
	})
	assertZfsReservation(suite.T(), dataset, true)
}

func (suite *ProvisionTestSuit) TestThickProvisionDataset() {
	dataset := provisionDataset(suite, "thick", map[string]string{
		provisioner.ParentDatasetParameter:   *parentDataset,
		provisioner.HostnameParameter:        "localhost",
		provisioner.TypeParameter:            "nfs",
		provisioner.SharePropertiesParameter: "rw,no_root_squash",
		provisioner.ReserveSpaceParameter:    "true",
	})
	assertZfsReservation(suite.T(), dataset, true)
}

func (suite *ProvisionTestSuit) TestThinProvisionDataset() {
	dataset := provisionDataset(suite, "thin", map[string]string{
		provisioner.ParentDatasetParameter:   *parentDataset,
		provisioner.HostnameParameter:        "localhost",
		provisioner.TypeParameter:            "nfs",
		provisioner.SharePropertiesParameter: "rw,no_root_squash",
		provisioner.ReserveSpaceParameter:    "false",
	})
	assertZfsReservation(suite.T(), dataset, false)
}

func provisionDataset(suite *ProvisionTestSuit, name string, parameters map[string]string) string {
	t := suite.T()
	pvName := suite.datasetPrefix + "_" + name
	fullDataset := *parentDataset + "/" + pvName
	datasetDirectory := "/" + fullDataset
	policy := v1.PersistentVolumeReclaimRetain
	options := controller.ProvisionOptions{
		PVName: pvName,
		PVC:    newClaim(resource.MustParse("10M"), []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce, v1.ReadOnlyMany}),
		StorageClass: &storagev1.StorageClass{
			Parameters:    parameters,
			ReclaimPolicy: &policy,
		},
	}

	_, _, err := suite.p.Provision(context.Background(), options)
	suite.createdDatasets = append(suite.createdDatasets, pvName)
	assert.NoError(t, err)
	require.DirExists(t, datasetDirectory)
	assertNfsExport(t, datasetDirectory)
	return fullDataset
}

func assertZfsReservation(t *testing.T, datasetName string, reserve bool) {
	fmt.Fprintln(os.Stderr, datasetName)

	dataset, err := gozfs.GetDataset(datasetName)
	assert.NoError(t, err)

	refreserved, err := dataset.GetProperty("refreservation")
	assert.NoError(t, err)

	refquota, err := dataset.GetProperty("refquota")
	assert.NoError(t, err)

	if reserve {
		assert.Equal(t, refquota, refreserved)
	} else {
		assert.Equal(t, "none", refreserved)
	}
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
