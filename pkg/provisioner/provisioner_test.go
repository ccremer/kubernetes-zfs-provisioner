package provisioner

import (
	"github.com/ccremer/kubernetes-zfs-provisioner/pkg/zfs"
	"github.com/stretchr/testify/mock"
)

type (
	zfsStub struct {
		mock.Mock
	}
)

func (z *zfsStub) GetDataset(name string) (*zfs.Dataset, error) {
	args := z.Called(name)
	return args.Get(0).(*zfs.Dataset), args.Error(1)
}

func (z *zfsStub) CreateDataset(name string, properties map[string]string) (*zfs.Dataset, error) {
	args := z.Called(name, properties)
	return args.Get(0).(*zfs.Dataset), args.Error(1)
}

func (z *zfsStub) DestroyDataset(dataset *zfs.Dataset, flag zfs.DestroyFlag) error {
	args := z.Called(dataset, flag)
	return args.Error(0)
}

func NewZFSProvisionerStub(stub *zfsStub) (*ZFSProvisioner, error) {
	return &ZFSProvisioner{zfs: stub}, nil
}
