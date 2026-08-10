package zfs

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// runLocal executes a command on the local host, for when the provisioner runs
// directly on the ZFS host (host is localhost or ZFS_EXEC_LOCAL=true). Arguments
// are passed as argv, so there is no shell and nothing to quote or inject.
func runLocal(ctx context.Context, binPrefix []string, args ...string) ([]byte, error) {
	if len(binPrefix) == 0 {
		return nil, fmt.Errorf("empty command")
	}
	full := append(append([]string{}, binPrefix[1:]...), args...)
	cmd := exec.CommandContext(ctx, binPrefix[0], full...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stdout.Bytes(), fmt.Errorf("local %q: %w: %s",
			strings.Join(append([]string{binPrefix[0]}, full...), " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

// isLocalHost reports whether host should be reached by local exec rather than SSH.
func (c sshConfig) isLocalHost(host string) bool {
	if c.forceLocal {
		return true
	}
	switch strings.ToLower(host) {
	case "", "localhost", "127.0.0.1", "::1":
		return true
	}
	return false
}
