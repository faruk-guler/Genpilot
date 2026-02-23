package ssh

import (
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// Client represents an SSH connection wrapper
type Client struct {
	client    *ssh.Client
	Config    *ssh.ClientConfig
	keepalive chan struct{} // signal to stop keepalive goroutine
}

// NewClient creates a new SSH client configuration with password auth
func NewClient(user, password, host string, port int, timeout time.Duration, hostKeyCallback ssh.HostKeyCallback) (*Client, error) {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	if hostKeyCallback == nil {
		var err error
		hostKeyCallback, err = getStrictHostKeyCallback()
		if err != nil {
			return nil, err
		}
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
			ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) (answers []string, err error) {
				answers = make([]string, len(questions))
				for i := range answers {
					answers[i] = password
				}
				return answers, nil
			}),
		},
		HostKeyCallback: hostKeyCallback,
		Timeout:         timeout,
	}

	return &Client{
		Config: config,
	}, nil
}

// NewClientWithKey creates a new SSH client configuration with key-based auth
func NewClientWithKey(user, keyPath string, host string, port int, timeout time.Duration, hostKeyCallback ssh.HostKeyCallback) (*Client, error) {
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}

	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	if hostKeyCallback == nil {
		var err error
		hostKeyCallback, err = getStrictHostKeyCallback()
		if err != nil {
			return nil, err
		}
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: hostKeyCallback,
		Timeout:         timeout,
	}

	return &Client{
		Config: config,
	}, nil
}

// Connect establishes the SSH connection
func (c *Client) Connect(addr string) error {
	client, err := ssh.Dial("tcp", addr, c.Config)
	if err != nil {
		return err
	}
	c.client = client
	return nil
}

// Close closes the connection and stops keepalive
func (c *Client) Close() error {
	c.StopKeepalive()
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// GetClient returns the underlying ssh.Client
func (c *Client) GetClient() *ssh.Client {
	return c.client
}

// RunCommand executes a single command and returns output
func (c *Client) RunCommand(cmd string) (string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

// PrepareShell creates a session and requests a PTY, but doesn't start the shell or assign pipes yet.
func (c *Client) PrepareShell(width, height int) (*ssh.Session, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return nil, err
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // enable echoing
		ssh.ICRNL:         1,     // Map CR to NL on input
		ssh.ONLCR:         1,     // Map NL to CR-NL on output
		ssh.ECHOCTL:       0,     // Don't echo control characters (like ^M)
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	if err := session.RequestPty("xterm", height, width, modes); err != nil {
		session.Close()
		return nil, err
	}

	return session, nil
}

// StartShell starts an interactive shell session (Legacy wrapper)
func (c *Client) StartShell(stdin io.Reader, stdout, stderr io.Writer, width, height int) (*ssh.Session, error) {
	session, err := c.PrepareShell(width, height)
	if err != nil {
		return nil, err
	}

	session.Stdin = stdin
	session.Stdout = stdout
	session.Stderr = stderr

	if err := session.Shell(); err != nil {
		session.Close()
		return nil, err
	}

	return session, nil
}

// ResizeTerminal resizes the current session terminal
func (c *Client) ResizeTerminal(session *ssh.Session, rows, cols int) error {
	if session == nil {
		return nil
	}
	return session.WindowChange(rows, cols)
}

// StartKeepalive starts sending keepalive packets at the given interval
func (c *Client) StartKeepalive(interval time.Duration) {
	if c.client == nil || interval <= 0 {
		return
	}

	c.keepalive = make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if c.client != nil {
					_, _, err := c.client.SendRequest("keepalive@openssh.com", true, nil)
					if err != nil {
						return // connection likely dead
					}
				}
			case <-c.keepalive:
				return
			}
		}
	}()
}

// StopKeepalive stops the keepalive goroutine
func (c *Client) StopKeepalive() {
	if c.keepalive != nil {
		select {
		case <-c.keepalive:
			// already closed
		default:
			close(c.keepalive)
		}
		c.keepalive = nil
	}
}

func (c *Client) IsConnected() bool {
	return c.client != nil
}

func getStrictHostKeyCallback() (ssh.HostKeyCallback, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return nil, err
	}
	knownHostsPath := filepath.Join(sshDir, "known_hosts")

	// Ensure file exists
	f, err := os.OpenFile(knownHostsPath, os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return nil, err
	}
	f.Close()

	// Use knownhosts package to check existing keys
	kh, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return nil, err
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := kh(hostname, remote, key)
		if err != nil {
			// Check if it's a KeyError and specifically if the host is unknown (Want is empty)
			if ke, ok := err.(*knownhosts.KeyError); ok && len(ke.Want) == 0 {
				f, ferr := os.OpenFile(knownHostsPath, os.O_APPEND|os.O_WRONLY, 0600)
				if ferr != nil {
					return ferr
				}
				defer f.Close()

				// Add the new host key to known_hosts
				line := knownhosts.Line([]string{hostname}, key)
				_, ferr = f.WriteString(line + "\n")
				return ferr
			}
			// If key is changed (MITM?) or other error, return it
			return err
		}
		return nil
	}, nil
}
