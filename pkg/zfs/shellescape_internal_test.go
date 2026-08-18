package zfs

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShellQuote(t *testing.T) {
	t.Parallel()

	cases := []struct{ in, want string }{
		{"", "''"},
		{"tank/volumes/pv-abc", "tank/volumes/pv-abc"},
		{"rw=@10.0.0.0/8", "rw=@10.0.0.0/8"},
		{"rw=@10.0.0.0/8 ro=@192.168.0.0/16", "'rw=@10.0.0.0/8 ro=@192.168.0.0/16'"},
		{"a;id", "'a;id'"},
		{"it's", `'it'\''s'`},
		{"$(id)", "'$(id)'"},
	}
	for _, c := range cases {
		require.Equal(t, c.want, shellQuote(c.in), "shellQuote(%q)", c.in)
	}
}

// TestShellQuoteRoundTrip proves that whatever we quote reaches a real POSIX
// shell as exactly one argument, byte-for-byte, no matter which metacharacters
// it contains.
func TestShellQuoteRoundTrip(t *testing.T) {
	t.Parallel()

	inputs := []string{
		"plain",
		"rw=@10.0.0.0/8 ro=@192.168.0.0/16",
		"two words",
		"semi;colon && echo rm",
		"$(id) `whoami`",
		"tab\there",
		"quote'inside",
	}
	for _, in := range inputs {
		out, err := exec.Command("/bin/sh", "-c", "printf %s "+shellQuote(in)).Output()
		require.NoErrorf(t, err, "sh for %q", in)
		require.Equalf(t, in, string(out), "round-trip for %q", in)
	}
}
