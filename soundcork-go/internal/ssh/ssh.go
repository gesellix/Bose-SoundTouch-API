package ssh

import (
	"bytes"
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"
)

// Client wraps an SSH client to perform operations on SoundTouch speakers.
type Client struct {
	Host string
	User string
}

// NewClient creates a new SSH client for the given host.
func NewClient(host string) *Client {
	return &Client{
		Host: host,
		User: "root",
	}
}

// getConfig returns the SSH client configuration.
func (c *Client) getConfig() *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: c.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(""), // Default password for SoundTouch root is often empty or not used with these settings
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
		Config: ssh.Config{
			KeyExchanges: []string{
				"diffie-hellman-group1-sha1",
				"diffie-hellman-group14-sha1",
				"ecdh-sha2-nistp256",
				"ecdh-sha2-nistp384",
				"ecdh-sha2-nistp521",
				"curve25519-sha256@libssh.org",
			},
		},
	}
}

// Run executes a command on the remote host and returns the output.
func (c *Client) Run(command string) (string, error) {
	config := c.getConfig()
	client, err := ssh.Dial("tcp", c.Host+":22", config)
	if err != nil {
		return "", fmt.Errorf("failed to dial: %v", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	var b bytes.Buffer
	session.Stdout = &b
	session.Stderr = &b
	if err := session.Run(command); err != nil {
		return b.String(), fmt.Errorf("failed to run command: %v", err)
	}
	return b.String(), nil
}

// UploadContent uploads the given content to a file on the remote host.
// It uses a simple approach: echoing the content into a file.
// For larger files, a proper SCP or SFTP implementation would be better.
func (c *Client) UploadContent(content []byte, remotePath string) error {
	config := c.getConfig()
	client, err := ssh.Dial("tcp", c.Host+":22", config)
	if err != nil {
		return fmt.Errorf("failed to dial: %v", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	// Use a pipe to write content to the remote command's stdin
	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %v", err)
	}

	go func() {
		defer stdin.Close()
		stdin.Write(content)
	}()

	// Read content from stdin and write to the remote file
	cmd := fmt.Sprintf("cat > %s", remotePath)
	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("failed to upload content: %v", err)
	}

	return nil
}
