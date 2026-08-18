package zfs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

type (
	// Interface abstracts the underlying ZFS host with the minimum functionality implemented
	Interface interface {
		GetDataset(name string, hostname string) (*Dataset, error)
		CreateDataset(name string, hostname string, properties map[string]string) (*Dataset, error)
		DestroyDataset(dataset *Dataset, flag DestroyFlag) error
		SetPermissions(dataset *Dataset) error
	}
	// Dataset is a representation of a ZFS dataset
	Dataset struct {
		Name       string
		Mountpoint string
		Hostname   string
	}
	DestroyFlag int

	// zfsImpl talks to ZFS hosts either over a native Go SSH connection or, when
	// running on the ZFS host itself (localhost or ZFS_EXEC_LOCAL=true), by local
	// exec. The target host is bound to the connection, not to a process-global
	// env var, and every zfs/chmod invocation is argument-quoted in Go (SSH) or
	// passed as argv (local) rather than parsed by a remote shell.
	zfsImpl struct {
		cfgOnce sync.Once
		cfg     sshConfig

		poolOnce sync.Once
		pool     *sshPool
		poolErr  error

		// runFn issues a command; nil means the default local/SSH dispatch. It
		// exists so the idempotency logic can be unit-tested with a fake runner.
		runFn func(ctx context.Context, host string, binPrefix []string, args ...string) ([]byte, error)
	}
)

const (
	DestroyRecursively DestroyFlag = 2
)

func NewInterface() *zfsImpl {
	return &zfsImpl{}
}

func (z *zfsImpl) config() sshConfig {
	z.cfgOnce.Do(func() { z.cfg = loadSSHConfig() })
	return z.cfg
}

// run dispatches a command to the local host or over SSH depending on the target
// host. The SSH pool is built lazily on first remote use, so a purely local
// deployment never needs SSH key material.
func (z *zfsImpl) run(ctx context.Context, host string, binPrefix []string, args ...string) ([]byte, error) {
	if z.runFn != nil {
		return z.runFn(ctx, host, binPrefix, args...)
	}
	cfg := z.config()
	if cfg.isLocalHost(host) {
		return runLocal(ctx, binPrefix, args...)
	}
	z.poolOnce.Do(func() { z.pool, z.poolErr = newSSHPool(cfg) })
	if z.poolErr != nil {
		return nil, z.poolErr
	}
	return z.pool.run(ctx, host, binPrefix, args...)
}

// presence reports whether a dataset exists. determinable is false when the
// check itself failed (e.g. the host is unreachable), so callers must not treat
// "not present" as "confirmed absent" unless determinable is true.
func (z *zfsImpl) presence(ctx context.Context, name, hostname string) (present, determinable bool) {
	if _, err := z.run(ctx, hostname, z.config().zfsBin, "list", "-H", "-o", "name", name); err == nil {
		return true, true
	} else if strings.Contains(err.Error(), "does not exist") {
		return false, true // ZFS confirmed the dataset is gone
	}
	return false, false // could not determine (transport error, permissions, ...)
}

func opCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 60*time.Second)
}

func (z *zfsImpl) GetDataset(name string, hostname string) (*Dataset, error) {
	ctx, cancel := opCtx()
	defer cancel()
	return z.getDataset(ctx, name, hostname)
}

func (z *zfsImpl) getDataset(ctx context.Context, name, hostname string) (*Dataset, error) {
	out, err := z.run(ctx, hostname, z.config().zfsBin, "list", "-Hp", "-o", "name,mountpoint", name)
	if err != nil {
		return nil, err
	}
	line := strings.TrimSpace(string(out))
	fields := strings.Split(line, "\t")
	if len(fields) < 2 {
		return nil, fmt.Errorf("unexpected 'zfs list' output for %s: %q", name, line)
	}
	return &Dataset{Name: fields[0], Mountpoint: fields[1], Hostname: hostname}, nil
}

func (z *zfsImpl) CreateDataset(name string, hostname string, properties map[string]string) (*Dataset, error) {
	ctx, cancel := opCtx()
	defer cancel()

	klog.V(3).InfoS("creating dataset", "name", name, "host", hostname)
	args := []string{"create"}
	for k, v := range properties {
		args = append(args, "-o", k+"="+v)
	}
	args = append(args, name)

	if _, err := z.run(ctx, hostname, z.config().zfsBin, args...); err != nil {
		// Idempotent: a retried Provision may find its own prior dataset. PV
		// names are unique (pvc-<uuid>), so a pre-existing dataset is never a
		// foreign collision; the desired end-state is already met. Only
		// converge when the dataset is confirmed present; otherwise surface the
		// original create error.
		if present, _ := z.presence(ctx, name, hostname); !present {
			return nil, err
		}
		klog.V(3).InfoS("dataset already exists, treating create as idempotent", "name", name, "host", hostname)
	}
	return z.getDataset(ctx, name, hostname)
}

func (z *zfsImpl) DestroyDataset(dataset *Dataset, flag DestroyFlag) error {
	if err := validateDataset(dataset); err != nil {
		return err
	}
	if flag != DestroyRecursively {
		return fmt.Errorf("programmer error: flag not implemented: %d", flag)
	}
	ctx, cancel := opCtx()
	defer cancel()

	if _, err := z.run(ctx, dataset.Hostname, z.config().zfsBin, "destroy", "-r", dataset.Name); err != nil {
		// Idempotent, but only when the dataset is *confirmed* absent. If we
		// cannot determine its state (e.g. the host is unreachable), surface the
		// error rather than reporting a phantom successful deletion.
		present, determinable := z.presence(ctx, dataset.Name, dataset.Hostname)
		if present || !determinable {
			return err
		}
		klog.V(3).InfoS("dataset already absent, treating destroy as idempotent", "name", dataset.Name, "host", dataset.Hostname)
	}
	return nil
}

func (z *zfsImpl) SetPermissions(dataset *Dataset) error {
	if err := validateDataset(dataset); err != nil {
		return err
	}
	if dataset.Mountpoint == "" {
		return fmt.Errorf("undefined mountpoint for dataset: %s", dataset.Name)
	}
	ctx, cancel := opCtx()
	defer cancel()

	// chmod g+w on the ZFS host (replaces docker/update-permissions.sh).
	cfg := z.config()
	if _, err := z.run(ctx, dataset.Hostname, cfg.chownBin, cfg.chmodArg, dataset.Mountpoint); err != nil {
		return fmt.Errorf("could not update permissions on '%s': %w", dataset.Hostname, err)
	}
	return nil
}

func validateDataset(dataset *Dataset) error {
	if dataset.Name == "" {
		return errors.New("undefined dataset name")
	}
	if dataset.Hostname == "" {
		return fmt.Errorf("required hostname parameter not given for dataset '%s'", dataset.Name)
	}
	return nil
}
