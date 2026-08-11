package zfs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kevinburke/ssh_config"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"k8s.io/klog/v2"
)

// sshConfig holds the connection settings, read once from the environment. The
// SSH key material and known_hosts are mounted into the pod (see the chart's
// ssh secret); everything else has a sensible default and can be overridden.
type sshConfig struct {
	mountPath  string
	envUser    string // ZFS_SSH_USER override (wins over ssh_config)
	envPort    string // ZFS_SSH_PORT override
	keyFile    string
	passphrase string
	knownHosts string
	tofu       bool // ZFS_SSH_HOSTKEY_TOFU=true: pin unknown host keys on first use
	requireTTY bool
	forceLocal bool // ZFS_EXEC_LOCAL=true: run zfs locally, never over SSH

	// sshCfg is the parsed ssh_config(5) mounted at <mountPath>/config, kept for
	// backward compatibility so existing per-host `User`/`Port` settings still
	// apply. nil when no config file is present.
	sshCfg *ssh_config.Config

	// Command prefixes, kept for parity with the previous shell wrappers.
	zfsBin   []string // default: sudo -H zfs
	chownBin []string // default: sudo -H chmod
	chmodArg string   // default: g+w
}

func loadSSHConfig() sshConfig {
	mount := env("ZFS_SSH_MOUNT_PATH", "/home/zfs/.ssh")
	c := sshConfig{
		mountPath:  mount,
		envUser:    os.Getenv("ZFS_SSH_USER"),
		envPort:    os.Getenv("ZFS_SSH_PORT"),
		keyFile:    os.Getenv("ZFS_SSH_KEY"),
		passphrase: os.Getenv("ZFS_SSH_KEY_PASSPHRASE"),
		knownHosts: env("ZFS_SSH_KNOWN_HOSTS", filepath.Join(mount, "known_hosts")),
		tofu:       os.Getenv("ZFS_SSH_HOSTKEY_TOFU") == "true",
		requireTTY: os.Getenv("ZFS_SSH_REQUIRETTY") == "true",
		forceLocal: os.Getenv("ZFS_EXEC_LOCAL") == "true",
		zfsBin:     strings.Fields(env("ZFS_BIN", "sudo -H zfs")),
		chownBin:   strings.Fields(env("ZFS_CHOWN_BIN", "sudo -H chmod")),
		chmodArg:   env("ZFS_MOD", "g+w"),
	}
	if f, err := os.Open(filepath.Join(mount, "config")); err == nil {
		defer f.Close()
		if parsed, perr := ssh_config.Decode(f); perr == nil {
			c.sshCfg = parsed
		} else {
			klog.Warningf("ignoring unparseable ssh_config at %s/config: %v", mount, perr)
		}
	}
	return c
}

// userFor resolves the SSH user for a host: the env override wins, then the
// ssh_config `User`, then "root".
func (c sshConfig) userFor(host string) string {
	if c.envUser != "" {
		return c.envUser
	}
	if c.sshCfg != nil {
		if u, _ := c.sshCfg.Get(host, "User"); u != "" {
			return u
		}
	}
	return "root"
}

// portFor resolves the SSH port for a host: env override, then ssh_config
// `Port`, then "22".
func (c sshConfig) portFor(host string) string {
	if c.envPort != "" {
		return c.envPort
	}
	if c.sshCfg != nil {
		if p, _ := c.sshCfg.Get(host, "Port"); p != "" {
			return p
		}
	}
	return "22"
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// sshPool hands out reusable SSH clients keyed by host. The target host comes
// from the StorageClass, so it is bound to a connection rather than to a
// process-global env var.
type sshPool struct {
	cfg     sshConfig
	signer  ssh.Signer
	hostKey ssh.HostKeyCallback

	mu      sync.Mutex
	clients map[string]*ssh.Client
}

func newSSHPool(cfg sshConfig) (*sshPool, error) {
	signer, err := loadSigner(cfg)
	if err != nil {
		return nil, err
	}
	hk, err := hostKeyCallback(cfg)
	if err != nil {
		return nil, err
	}
	return &sshPool{cfg: cfg, signer: signer, hostKey: hk, clients: map[string]*ssh.Client{}}, nil
}

func loadSigner(cfg sshConfig) (ssh.Signer, error) {
	path := cfg.keyFile
	if path == "" {
		var err error
		if path, err = discoverKey(cfg.mountPath); err != nil {
			return nil, err
		}
	}
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read ssh key %s: %w", path, err)
	}
	if cfg.passphrase != "" {
		return ssh.ParsePrivateKeyWithPassphrase(key, []byte(cfg.passphrase))
	}
	return ssh.ParsePrivateKey(key)
}

// discoverKey picks the first parseable private key in the mount directory,
// preferring the conventional names.
func discoverKey(dir string) (string, error) {
	for _, name := range []string{"id_ed25519", "id_ecdsa", "id_rsa"} {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("no SSH key found in %s under a standard name (id_ed25519, id_ecdsa, id_rsa); set ZFS_SSH_KEY to use another path", dir)
}

// hostKeyCallback verifies remote host keys against the mounted known_hosts.
// A key that does not match a recorded one is always rejected (possible MITM).
// When ZFS_SSH_HOSTKEY_TOFU=true, a host that is not yet recorded is pinned on
// first use and appended to known_hosts: the native equivalent of ssh-keyscan,
// so no external tooling is needed.
func hostKeyCallback(cfg sshConfig) (ssh.HostKeyCallback, error) {
	var verify ssh.HostKeyCallback
	if _, err := os.Stat(cfg.knownHosts); err == nil {
		if verify, err = knownhosts.New(cfg.knownHosts); err != nil {
			return nil, fmt.Errorf("parse known_hosts %s: %w", cfg.knownHosts, err)
		}
	} else if !cfg.tofu {
		return nil, fmt.Errorf("known_hosts %s not found; provide it, or set ZFS_SSH_HOSTKEY_TOFU=true to pin host keys on first use: %w", cfg.knownHosts, err)
	}

	if !cfg.tofu {
		return verify, nil
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if verify != nil {
			err := verify(hostname, remote, key)
			if err == nil {
				return nil
			}
			var ke *knownhosts.KeyError
			if errors.As(err, &ke) && len(ke.Want) > 0 {
				return err // host is known but presented a different key
			}
		}
		recordHostKey(cfg.knownHosts, hostname, key)
		return nil
	}, nil
}

// recordHostKey pins a previously unseen host key by appending it to known_hosts
// in the standard format. If the file is not writable (e.g. a read-only mount)
// the line is logged so it can be added by hand.
func recordHostKey(path, hostname string, key ssh.PublicKey) {
	line := strings.TrimSpace(knownhosts.Line([]string{knownhosts.Normalize(hostname)}, key))
	klog.Warningf("pinning new host key (TOFU) for %s: %s", hostname, line)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		klog.Warningf("could not persist host key to %s; add the line above manually: %v", path, err)
		return
	}
	defer f.Close()
	_, _ = f.WriteString(line + "\n")
}

func (p *sshPool) client(host string) (*ssh.Client, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, ok := p.clients[host]; ok {
		// Cheap liveness check; re-dial if the connection went away.
		if _, _, err := c.SendRequest("keepalive@openssh.com", true, nil); err == nil {
			return c, nil
		}
		_ = c.Close()
		delete(p.clients, host)
	}
	cfg := &ssh.ClientConfig{
		User:            p.cfg.userFor(host),
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(p.signer)},
		HostKeyCallback: p.hostKey,
		Timeout:         15 * time.Second,
	}
	c, err := ssh.Dial("tcp", net.JoinHostPort(host, p.cfg.portFor(host)), cfg)
	if err != nil {
		return nil, fmt.Errorf("ssh dial %s: %w", host, err)
	}
	p.clients[host] = c
	return c, nil
}

// run executes a command on host. Each argument is shell-quoted in Go so spaces
// and metacharacters cannot be re-split by the remote shell. binPrefix words
// (e.g. "sudo -H zfs") are passed verbatim; only args are quoted.
func (p *sshPool) run(ctx context.Context, host string, binPrefix []string, args ...string) ([]byte, error) {
	client, err := p.client(host)
	if err != nil {
		return nil, err
	}
	sess, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	defer sess.Close()

	if p.cfg.requireTTY {
		modes := ssh.TerminalModes{ssh.ECHO: 0, ssh.TTY_OP_ISPEED: 14400, ssh.TTY_OP_OSPEED: 14400}
		if err := sess.RequestPty("xterm", 200, 80, modes); err != nil {
			return nil, fmt.Errorf("request pty: %w", err)
		}
	}

	parts := make([]string, 0, len(binPrefix)+len(args))
	parts = append(parts, binPrefix...)
	for _, a := range args {
		parts = append(parts, shellQuote(a))
	}
	cmd := strings.Join(parts, " ")

	var stdout, stderr bytes.Buffer
	sess.Stdout = &stdout
	sess.Stderr = &stderr

	done := make(chan error, 1)
	go func() { done <- sess.Run(cmd) }()
	select {
	case <-ctx.Done():
		_ = sess.Signal(ssh.SIGKILL)
		return nil, ctx.Err()
	case err = <-done:
	}
	if err != nil {
		return stdout.Bytes(), fmt.Errorf("remote %q on %s: %w: %s", cmd, host, err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}
