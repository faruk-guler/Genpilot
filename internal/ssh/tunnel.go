package ssh

import (
	"context"
	"fmt"
	"io"
	"net"
)

// Tunnel represents a local port forwarding tunnel
type Tunnel struct {
	ID         string
	LocalPort  int
	RemoteHost string
	RemotePort int
	listener   net.Listener
	ctx        context.Context
	cancel     context.CancelFunc
}

// StartLocalForward starts local port forwarding
func (c *Client) StartLocalForward(id string, localPort int, remoteHost string, remotePort int) (*Tunnel, error) {
	if c.client == nil {
		return nil, fmt.Errorf("ssh client not connected")
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", localPort))
	if err != nil {
		return nil, fmt.Errorf("failed to listen on local port %d: %w", localPort, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	tunnel := &Tunnel{
		ID:         id,
		LocalPort:  localPort,
		RemoteHost: remoteHost,
		RemotePort: remotePort,
		listener:   listener,
		ctx:        ctx,
		cancel:     cancel,
	}

	go func() {
		for {
			localConn, err := listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return // Tunnel closed intentionally
				default:
					continue
				}
			}

			go c.handleTunnelConnection(localConn, remoteHost, remotePort)
		}
	}()

	return tunnel, nil
}

func (c *Client) handleTunnelConnection(localConn net.Conn, remoteHost string, remotePort int) {
	defer localConn.Close()

	remoteConn, err := c.client.Dial("tcp", fmt.Sprintf("%s:%d", remoteHost, remotePort))
	if err != nil {
		return
	}
	defer remoteConn.Close()

	// Copy local to remote
	go func() {
		io.Copy(remoteConn, localConn)
	}()

	// Copy remote to local
	io.Copy(localConn, remoteConn)
}

// Stop closes the tunnel listener
func (t *Tunnel) Stop() error {
	t.cancel()
	if t.listener != nil {
		return t.listener.Close()
	}
	return nil
}
