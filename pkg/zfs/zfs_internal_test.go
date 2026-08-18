package zfs

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeRunner lets the idempotency logic be exercised without a real ZFS host.
// It dispatches on the zfs subcommand (create / destroy / list).
type fakeRunner struct {
	onCreate  error
	onDestroy error
	listOut   string
	listErr   error
	calls     []string
}

func (f *fakeRunner) run(_ context.Context, _ string, _ []string, args ...string) ([]byte, error) {
	if len(args) == 0 {
		return nil, errors.New("no args")
	}
	f.calls = append(f.calls, args[0])
	switch args[0] {
	case "create":
		return nil, f.onCreate
	case "destroy":
		return nil, f.onDestroy
	case "list":
		return []byte(f.listOut), f.listErr
	case "set":
		return nil, nil
	}
	return nil, errors.New("unexpected: " + args[0])
}

func newImpl(f *fakeRunner) *zfsImpl {
	return &zfsImpl{runFn: f.run}
}

func TestGetDatasetParsing(t *testing.T) {
	t.Parallel()
	z := newImpl(&fakeRunner{listOut: "tank/volumes/pv-a\t/tank/volumes/pv-a\n"})
	ds, err := z.GetDataset("tank/volumes/pv-a", "h")
	require.NoError(t, err)
	require.Equal(t, "tank/volumes/pv-a", ds.Name)
	require.Equal(t, "/tank/volumes/pv-a", ds.Mountpoint)
	require.Equal(t, "h", ds.Hostname)
}

func TestCreateIdempotentWhenAlreadyPresent(t *testing.T) {
	t.Parallel()
	// create fails "already exists" but the dataset is present -> converge.
	f := &fakeRunner{
		onCreate: errors.New("cannot create 'x': dataset already exists"),
		listOut:  "tank/volumes/pv-a\t/tank/volumes/pv-a\n",
	}
	z := newImpl(f)
	ds, err := z.CreateDataset("tank/volumes/pv-a", "h", map[string]string{"refquota": "10"})
	require.NoError(t, err)
	require.Equal(t, "/tank/volumes/pv-a", ds.Mountpoint)
}

func TestCreateReturnsErrorWhenTrulyAbsent(t *testing.T) {
	t.Parallel()
	// create fails and the dataset does NOT exist -> surface the create error.
	f := &fakeRunner{
		onCreate: errors.New("cannot create parent: permission denied"),
		listErr:  errors.New("cannot open 'x': dataset does not exist"),
	}
	z := newImpl(f)
	_, err := z.CreateDataset("tank/volumes/pv-a", "h", nil)
	require.ErrorContains(t, err, "permission denied")
}

func TestDestroyIdempotentWhenConfirmedAbsent(t *testing.T) {
	t.Parallel()
	f := &fakeRunner{
		onDestroy: errors.New("cannot open 'x': dataset does not exist"),
		listErr:   errors.New("cannot open 'x': dataset does not exist"),
	}
	z := newImpl(f)
	require.NoError(t, z.DestroyDataset(&Dataset{Name: "x", Hostname: "h"}, DestroyRecursively))
}

// TestDestroyDoesNotMaskTransportError is the regression guard: a destroy that
// fails while the host is unreachable must NOT be reported as a successful
// (phantom) deletion.
func TestDestroyDoesNotMaskTransportError(t *testing.T) {
	t.Parallel()
	f := &fakeRunner{
		onDestroy: errors.New("ssh dial h: connection refused"),
		listErr:   errors.New("ssh dial h: connection refused"), // cannot determine
	}
	z := newImpl(f)
	err := z.DestroyDataset(&Dataset{Name: "x", Hostname: "h"}, DestroyRecursively)
	require.Error(t, err, "must not report success when the dataset state is unknown")
	require.ErrorContains(t, err, "connection refused")
}

func TestDestroyReturnsErrorWhenStillPresent(t *testing.T) {
	t.Parallel()
	f := &fakeRunner{
		onDestroy: errors.New("cannot destroy 'x': dataset is busy"),
		listOut:   "x\t/x\n", // list succeeds -> still present
	}
	z := newImpl(f)
	err := z.DestroyDataset(&Dataset{Name: "x", Hostname: "h"}, DestroyRecursively)
	require.ErrorContains(t, err, "is busy")
}
