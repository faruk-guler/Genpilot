package config

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
)

// HostKeyResult indicates the result of a host key check
type HostKeyResult int

const (
	HostKeyNew     HostKeyResult = iota // Host not seen before
	HostKeyMatch                        // Key matches saved fingerprint
	HostKeyChanged                      // Key changed (potential MITM)
)

// KnownHostsManager manages SSH host key fingerprints
type KnownHostsManager struct {
	mu       sync.Mutex
	filePath string
	hosts    map[string]string // host -> fingerprint
}

// NewKnownHostsManager creates a new known hosts manager
func NewKnownHostsManager() (*KnownHostsManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configDir := filepath.Join(homeDir, ".genpilot")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, err
	}

	km := &KnownHostsManager{
		filePath: filepath.Join(configDir, "known_hosts"),
		hosts:    make(map[string]string),
	}

	km.load()
	return km, nil
}

// Fingerprint returns the SHA-256 fingerprint of an SSH public key
func Fingerprint(key ssh.PublicKey) string {
	hash := sha256.Sum256(key.Marshal())
	return base64.StdEncoding.EncodeToString(hash[:])
}

// Check verifies a host's public key against stored fingerprints
func (km *KnownHostsManager) Check(host string, key ssh.PublicKey) HostKeyResult {
	km.mu.Lock()
	defer km.mu.Unlock()

	fp := Fingerprint(key)
	stored, exists := km.hosts[host]

	if !exists {
		return HostKeyNew
	}
	if stored == fp {
		return HostKeyMatch
	}
	return HostKeyChanged
}

// Add stores a host's public key fingerprint
func (km *KnownHostsManager) Add(host string, key ssh.PublicKey) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	fp := Fingerprint(key)
	km.hosts[host] = fp
	return km.save()
}

// GetFingerprint returns the stored fingerprint for a host
func (km *KnownHostsManager) GetFingerprint(host string) (string, bool) {
	km.mu.Lock()
	defer km.mu.Unlock()

	fp, ok := km.hosts[host]
	return fp, ok
}

// HostKeyCallback returns an ssh.HostKeyCallback that uses a user-provided prompt function
// promptFn should return true to accept the key, false to reject
func (km *KnownHostsManager) HostKeyCallback(promptFn func(host string, keyType string, fingerprint string, result HostKeyResult) bool) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		// Normalize hostname
		host := hostname
		if h, _, err := net.SplitHostPort(hostname); err == nil {
			host = h
		}

		fp := Fingerprint(key)
		keyType := key.Type()
		result := km.Check(host, key)

		switch result {
		case HostKeyMatch:
			return nil
		case HostKeyNew:
			if promptFn(host, keyType, fp, HostKeyNew) {
				km.Add(host, key)
				return nil
			}
			return fmt.Errorf("host key rejected by user")
		case HostKeyChanged:
			if promptFn(host, keyType, fp, HostKeyChanged) {
				km.Add(host, key)
				return nil
			}
			return fmt.Errorf("host key changed — connection refused")
		}

		return fmt.Errorf("unknown host key result")
	}
}

func (km *KnownHostsManager) load() {
	f, err := os.Open(km.filePath)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			km.hosts[parts[0]] = parts[1]
		}
	}
}

func (km *KnownHostsManager) save() error {
	f, err := os.Create(km.filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	w.WriteString("# Genpilot Known Hosts\n")
	for host, fp := range km.hosts {
		fmt.Fprintf(w, "%s %s\n", host, fp)
	}
	return w.Flush()
}
