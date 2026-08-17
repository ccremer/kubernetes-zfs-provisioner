package zfs

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// newHostKey returns a fresh, unique SSH public key to stand in for a host key.
func newHostKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	return signer.PublicKey()
}

func writeKnownHosts(t *testing.T, path, host string, key ssh.PublicKey) {
	t.Helper()
	line := knownhosts.Line([]string{knownhosts.Normalize(host)}, key)
	if err := os.WriteFile(path, []byte(line+"\n"), 0o600); err != nil {
		t.Fatalf("write known_hosts: %v", err)
	}
}

var testAddr = &net.TCPAddr{IP: net.IPv4(10, 0, 0, 1), Port: 22}

// Strict mode (the default): only a key matching the recorded one is accepted.
func TestHostKeyCallback_StrictRejectsChangedAndUnknown(t *testing.T) {
	kh := filepath.Join(t.TempDir(), "known_hosts")
	recorded, changed := newHostKey(t), newHostKey(t)
	const host = "storage-1:22"
	writeKnownHosts(t, kh, host, recorded)

	cb, err := hostKeyCallback(sshConfig{knownHosts: kh, tofu: false})
	if err != nil {
		t.Fatalf("hostKeyCallback: %v", err)
	}

	if err := cb(host, testAddr, recorded); err != nil {
		t.Errorf("matching key rejected: %v", err)
	}
	if err := cb(host, testAddr, changed); err == nil {
		t.Error("changed host key accepted; want rejection (possible MITM)")
	}
	if err := cb("other-host:22", testAddr, recorded); err == nil {
		t.Error("unknown host accepted; want rejection in strict mode")
	}
}

// Strict mode refuses to start when known_hosts is absent, rather than connecting blindly.
func TestHostKeyCallback_StrictMissingKnownHostsFails(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	if _, err := hostKeyCallback(sshConfig{knownHosts: missing, tofu: false}); err == nil {
		t.Fatal("expected error when known_hosts is missing and TOFU is off")
	}
}

// TOFU mode: an unrecorded host is pinned on first use, but a recorded host that
// presents a different key is still rejected.
func TestHostKeyCallback_TOFUPinsUnknownButRejectsChanged(t *testing.T) {
	kh := filepath.Join(t.TempDir(), "known_hosts")
	recorded, changed, fresh := newHostKey(t), newHostKey(t), newHostKey(t)
	const known = "known-host:22"
	writeKnownHosts(t, kh, known, recorded)

	cb, err := hostKeyCallback(sshConfig{knownHosts: kh, tofu: true})
	if err != nil {
		t.Fatalf("hostKeyCallback: %v", err)
	}

	if err := cb(known, testAddr, recorded); err != nil {
		t.Errorf("recorded host with matching key rejected: %v", err)
	}
	if err := cb(known, testAddr, changed); err == nil {
		t.Error("recorded host with changed key accepted; TOFU must still reject it")
	}

	const newHost = "new-host:22"
	if err := cb(newHost, testAddr, fresh); err != nil {
		t.Errorf("TOFU rejected an unknown host on first use: %v", err)
	}
	data, err := os.ReadFile(kh)
	if err != nil {
		t.Fatalf("read known_hosts: %v", err)
	}
	if !strings.Contains(string(data), "new-host") {
		t.Errorf("TOFU did not persist the pinned host key; known_hosts:\n%s", data)
	}
}
