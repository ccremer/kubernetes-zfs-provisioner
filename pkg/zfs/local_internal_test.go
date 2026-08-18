package zfs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRunLocalArgvIntegrity proves the local path passes each argument as its own
// argv entry: spaces are preserved and shell metacharacters are never evaluated,
// because there is no shell involved at all.
func TestRunLocalArgvIntegrity(t *testing.T) {
	t.Parallel()

	// printf '%s' <arg> echoes exactly one argument back.
	got, err := runLocal(t.Context(), []string{"printf", "%s"}, "rw=@10.0.0.0/8 ro=@192.168.0.0/16")
	require.NoError(t, err)
	require.Equal(t, "rw=@10.0.0.0/8 ro=@192.168.0.0/16", string(got))

	got, err = runLocal(t.Context(), []string{"printf", "%s"}, "$(id) `whoami`")
	require.NoError(t, err)
	require.Equal(t, "$(id) `whoami`", string(got), "metacharacters must be literal, not executed")
}

func TestIsLocalHost(t *testing.T) {
	t.Parallel()

	c := sshConfig{}
	for _, h := range []string{"", "localhost", "LOCALHOST", "127.0.0.1", "::1"} {
		require.Truef(t, c.isLocalHost(h), "%q should be local", h)
	}
	for _, h := range []string{"zfs-host", "10.0.0.5", "host.example.com"} {
		require.Falsef(t, c.isLocalHost(h), "%q should be remote", h)
	}

	forced := sshConfig{forceLocal: true}
	require.True(t, forced.isLocalHost("any-remote-host"), "ZFS_EXEC_LOCAL forces local")
}
