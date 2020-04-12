package zfs

import (
	"fmt"
	gozfs "github.com/mistifyio/go-zfs"
)

type (
	// Interface abstracts the underlying ZFS library with the minimum functionality implemented
	Interface interface {
		GetDataset(name string) (*Dataset, error)
		CreateDataset(name string, properties map[string]string) (*Dataset, error)
		DestroyDataset(dataset *Dataset, flag DestroyFlag) error
	}
	// Dataset is a representation of a ZFS dataset
	Dataset struct {
		datasetImpl *gozfs.Dataset

		Name       string
		Mountpoint string
	}
	DestroyFlag int
	zfsImpl     struct{}
)

const (
	DestroyRecursively DestroyFlag = 2
)

func (z *zfsImpl) GetDataset(name string) (*Dataset, error) {
	dataset, err := gozfs.GetDataset(name)
	if err != nil {
		return nil, err
	}
	return &Dataset{
		datasetImpl: dataset,
		Name:        dataset.Name,
		Mountpoint:  dataset.Mountpoint,
	}, err
}

func (z *zfsImpl) CreateDataset(name string, properties map[string]string) (*Dataset, error) {
	dataset, err := gozfs.CreateFilesystem(name, properties)
	if err != nil {
		return nil, err
	}
	return &Dataset{
		datasetImpl: dataset,
		Name:        dataset.Name,
		Mountpoint:  dataset.Mountpoint,
	}, err
}

func (z *zfsImpl) DestroyDataset(dataset *Dataset, flag DestroyFlag) error {
	if dataset.datasetImpl == nil {
		ds, err := gozfs.GetDataset(dataset.Name)
		if err != nil {
			return err
		}
		dataset.datasetImpl = ds
	}
	var destrFlag gozfs.DestroyFlag
	switch flag {
	case DestroyRecursively:
		destrFlag = gozfs.DestroyRecursive
		break
	default:
		return fmt.Errorf("programmer error: flag not implemented: %d", flag)
	}
	return dataset.datasetImpl.Destroy(destrFlag)
}

func NewInterface() *zfsImpl {
	return &zfsImpl{}
}
