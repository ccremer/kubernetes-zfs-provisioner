package zfs

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsNotFound(t *testing.T) {
	t.Parallel()
	require.True(t, IsNotFound(fmt.Errorf("cannot open 'x': dataset does not exist")))
	require.True(t, IsNotFound(fmt.Errorf("cannot open 'x': no such dataset")))
	require.True(t, IsNotFound(&RunError{Ran: true, Err: fmt.Errorf("exit 1"), Stderr: "dataset does not exist"}))
	require.False(t, IsNotFound(&RunError{Ran: false, Err: fmt.Errorf("dataset does not exist")}), "transport must not look like not-found")
	require.False(t, IsNotFound(fmt.Errorf("permission denied")))
	require.False(t, IsNotFound(fmt.Errorf("ssh dial h: connection refused")))
}

func TestIsTransient(t *testing.T) {
	t.Parallel()
	require.True(t, IsTransient(fmt.Errorf("ssh dial nas: connection refused")))
	require.True(t, IsTransient(fmt.Errorf("dataset is busy")))
	require.True(t, IsTransient(context.DeadlineExceeded))
	require.True(t, IsTransient(&RunError{Ran: false, Err: fmt.Errorf("dial")}))
	require.False(t, IsTransient(fmt.Errorf("cannot create: permission denied")))
	require.False(t, IsTransient(fmt.Errorf("invalid property 'foo'")))
}

func TestParseAvailableBytes(t *testing.T) {
	t.Parallel()
	n, ok := ParseAvailableBytes("1048576")
	require.True(t, ok)
	require.Equal(t, int64(1048576), n)
	_, ok = ParseAvailableBytes("none")
	require.False(t, ok)
}
