package provisioner

import (
	"github.com/stretchr/testify/mock"

	"github.com/ccremer/kubernetes-zfs-provisioner/pkg/zfs"
)

type (
	zfsStub struct {
		mock.Mock
	}
)

func (z *zfsStub) GetDataset(name string, hostname string) (*zfs.Dataset, error) {
	args := z.Called(name, hostname)
	return args.Get(0).(*zfs.Dataset), args.Error(1)
}

func (z *zfsStub) CreateDataset(name string, hostname string, properties map[string]string) (*zfs.Dataset, error) {
	args := z.Called(name, properties)
	return args.Get(0).(*zfs.Dataset), args.Error(1)
}

func (z *zfsStub) DestroyDataset(dataset *zfs.Dataset, flag zfs.DestroyFlag) error {
	args := z.Called(dataset, flag)
	return args.Error(0)
}

func (z *zfsStub) SetPermissions(dataset *zfs.Dataset) error {
	args := z.Called(dataset)
	return args.Error(0)
}

func (z *zfsStub) SetProperty(dataset *zfs.Dataset, key, value string) error {
	args := z.Called(dataset, key, value)
	return args.Error(0)
}

func (z *zfsStub) GetProperty(name, hostname, key string) (string, error) {
	args := z.Called(name, hostname, key)
	return args.String(0), args.Error(1)
}

func (z *zfsStub) ListSnapshots(dataset *zfs.Dataset) ([]string, error) {
	args := z.Called(dataset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func NewZFSProvisionerStub(stub *zfsStub) (*ZFSProvisioner, error) {
	return &ZFSProvisioner{
		zfs:          stub,
		InstanceName: "test",
	}, nil
}
