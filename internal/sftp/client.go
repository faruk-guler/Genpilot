package sftp

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// Client represents an SFTP connection wrapper
type Client struct {
	client *sftp.Client
}

// NewClient creates a new SFTP client from an existing SSH client
func NewClient(sshClient *ssh.Client) (*Client, error) {
	client, err := sftp.NewClient(sshClient)
	if err != nil {
		return nil, err
	}

	return &Client{
		client: client,
	}, nil
}

// Close closes the SFTP connection
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// ListDirectory returns a list of files in the given directory
func (c *Client) ListDirectory(dirPath string) ([]os.FileInfo, error) {
	return c.client.ReadDir(dirPath)
}

// Upload uploads a local file to the remote server
func (c *Client) Upload(localPath, remotePath string) error {
	localFile, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer localFile.Close()

	remoteFile, err := c.client.Create(remotePath)
	if err != nil {
		return err
	}
	defer remoteFile.Close()

	_, err = io.Copy(remoteFile, localFile)
	return err
}

// Download downloads a remote file to the local machine
func (c *Client) Download(remotePath, localPath string) error {
	remoteFile, err := c.client.Open(remotePath)
	if err != nil {
		return err
	}
	defer remoteFile.Close()

	localFile, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer localFile.Close()

	_, err = io.Copy(localFile, remoteFile)
	return err
}

// Getwd returns the current working directory
func (c *Client) Getwd() (string, error) {
	return c.client.Getwd()
}

// Join joins any number of path elements into a single path, adding a separating slash if necessary.
func (c *Client) Join(elem ...string) string {
	return path.Join(elem...)
}

// Mkdir creates a new directory on the remote server
func (c *Client) Mkdir(remotePath string) error {
	return c.client.Mkdir(remotePath)
}

// MkdirAll creates a directory tree on the remote server
func (c *Client) MkdirAll(remotePath string) error {
	return c.client.MkdirAll(remotePath)
}

// Rename renames a file or directory on the remote server
func (c *Client) Rename(oldPath, newPath string) error {
	return c.client.Rename(oldPath, newPath)
}

// Remove removes a file or directory on the remote server
func (c *Client) Remove(remotePath string) error {
	return c.client.Remove(remotePath)
}

// Stat returns file info for a remote path
func (c *Client) Stat(remotePath string) (os.FileInfo, error) {
	return c.client.Stat(remotePath)
}

// GetSFTPClient returns the underlying sftp.Client for direct use
func (c *Client) GetSFTPClient() *sftp.Client {
	return c.client
}

// Chmod changes the permissions of a remote file
func (c *Client) Chmod(remotePath string, mode os.FileMode) error {
	return c.client.Chmod(remotePath, mode)
}

// DownloadDirectory recursively downloads a remote directory to a local path
func (c *Client) DownloadDirectory(remotePath, localPath string) error {
	// Create the local directory
	if err := os.MkdirAll(localPath, 0755); err != nil {
		return fmt.Errorf("create local dir: %w", err)
	}

	// List remote directory contents
	entries, err := c.client.ReadDir(remotePath)
	if err != nil {
		return fmt.Errorf("read remote dir: %w", err)
	}

	for _, entry := range entries {
		remoteItem := path.Join(remotePath, entry.Name())
		localItem := filepath.Join(localPath, entry.Name())

		if entry.IsDir() {
			// Recursively download subdirectory
			if err := c.DownloadDirectory(remoteItem, localItem); err != nil {
				return err
			}
		} else {
			// Download file
			if err := c.Download(remoteItem, localItem); err != nil {
				return fmt.Errorf("download %s: %w", entry.Name(), err)
			}
		}
	}

	return nil
}

// UploadDirectory recursively uploads a local directory to a remote path
func (c *Client) UploadDirectory(localPath, remotePath string) error {
	// Create the remote directory
	c.client.MkdirAll(remotePath)

	// Walk local directory
	entries, err := os.ReadDir(localPath)
	if err != nil {
		return fmt.Errorf("read local dir: %w", err)
	}

	for _, entry := range entries {
		localItem := filepath.Join(localPath, entry.Name())
		remoteItem := path.Join(remotePath, entry.Name())

		if entry.IsDir() {
			// Recursively upload subdirectory
			if err := c.UploadDirectory(localItem, remoteItem); err != nil {
				return err
			}
		} else {
			// Upload file
			if err := c.Upload(localItem, remoteItem); err != nil {
				return fmt.Errorf("upload %s: %w", entry.Name(), err)
			}
		}
	}

	return nil
}

// RemoveDirectory recursively removes a remote directory
func (c *Client) RemoveDirectory(remotePath string) error {
	entries, err := c.client.ReadDir(remotePath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		itemPath := path.Join(remotePath, entry.Name())
		if entry.IsDir() {
			if err := c.RemoveDirectory(itemPath); err != nil {
				return err
			}
		} else {
			if err := c.client.Remove(itemPath); err != nil {
				return err
			}
		}
	}

	return c.client.RemoveDirectory(remotePath)
}
