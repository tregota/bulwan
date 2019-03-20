package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

func getAuthMethod(privatekey string) (ssh.AuthMethod, error) {
	key, err := ssh.ParsePrivateKey([]byte(CleanPrivateKey(privatekey)))
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeys(key), nil
}

// Endpoint contains all that is needed to connect to an endpoint
type Endpoint struct {
	Host          string
	Port          int
	PublicKey     string
	PublicKeyType string
}

func (endpoint *Endpoint) String() string {
	return fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
}

func (endpoint *Endpoint) getPublicKey() (ssh.PublicKey, error) {
	if endpoint.PublicKey == "" {
		return nil, errors.New("no public key found")
	}
	if endpoint.PublicKeyType == "" {
		return nil, errors.New("no public key type found")
	}
	_, _, pubKey, _, _, err := ssh.ParseKnownHosts([]byte(fmt.Sprintf("%s %s %s", endpoint.Host, endpoint.PublicKeyType, endpoint.PublicKey))) // marker, hosts, pubKey, comment, rest, err
	if err != nil {
		return nil, err
	}
	return pubKey, nil
}

// SSHConn is an established SSH connection
type SSHConn struct {
	Server     *Endpoint
	Connection *ssh.Client
}

// SSHDial sets up a connection to the endpoint using the given username and given private key file path
func (endpoint *Endpoint) SSHDial(username string, privatekey string) (*SSHConn, error) {

	authMethod, err := getAuthMethod(privatekey)
	if err != nil {
		return nil, err
	}

	hostkeycallback := ssh.InsecureIgnoreHostKey()
	if endpoint.PublicKey != "" {
		publicKey, err := endpoint.getPublicKey()
		if err != nil {
			return nil, err
		}
		hostkeycallback = ssh.FixedHostKey(publicKey)
	}

	sshConfig := &ssh.ClientConfig{
		User:            username,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: hostkeycallback,
		Timeout:         10 * time.Second,
	}

	conn, err := ssh.Dial("tcp", endpoint.String(), sshConfig)
	if err != nil {
		return nil, err
	}

	sshTunnel := &SSHConn{
		Server:     endpoint,
		Connection: conn,
	}
	return sshTunnel, nil
}

func copyConn(writer, reader net.Conn) {
	defer writer.Close()
	defer reader.Close()
	_, err := io.Copy(writer, reader)
	if err != nil {
		fmt.Printf("Error: forward io.Copy - %v\n", err)
	}
}
func forward(sourceConn net.Conn, destinationEndpoint *Endpoint) {

	destinationConn, err := net.DialTimeout("tcp", destinationEndpoint.String(), 10*time.Second)
	if err != nil {
		fmt.Printf("Error: %s:%d forward dial - %v", destinationEndpoint.Host, destinationEndpoint.Port, err)
		return
	}

	go copyConn(sourceConn, destinationConn)
	copyConn(destinationConn, sourceConn)
}

// ReverseTunnelListen binds a port on the remote server and returns a listener
func (conn *SSHConn) ReverseTunnelListen(sourceport int) (net.Listener, error) {
	sourceEndpoint := &Endpoint{
		Host: "0.0.0.0",
		Port: sourceport,
	}
	return conn.Connection.Listen("tcp", sourceEndpoint.String())
}

// ReverseTunnelForceListen binds a port on the remote server even if already bound by username and returns a listener
func (conn *SSHConn) ReverseTunnelForceListen(sourceport int, username string) (net.Listener, error) {
	for {
		listener, err := conn.ReverseTunnelListen(sourceport)
		if err != nil && strings.Contains(err.Error(), "tcpip-forward request denied by peer") {
			session, err := conn.Connection.NewSession()
			if err != nil {
				// return to reconnect and try again
				return nil, fmt.Errorf("%s:%d listen - unable to bind port - tcpip-forward denied and failed to open kill session - %v", conn.Server.Host, sourceport, err)
			}
			err = session.Run(fmt.Sprintf("pkill -o -u %s sshd", username)) // kill oldest sshd process owned by user
			session.Close()
			if err != nil {
				// kill failed, probably because it killed it's own connection which means there were no others
				return nil, fmt.Errorf("%s:%d listen - unable to bind port - tcpip-forward denied and no remnant ssh connections found", conn.Server.Host, sourceport)
			}
			continue
		} else if err != nil {
			return nil, fmt.Errorf("%s:%d listen - %v", conn.Server.Host, sourceport, err)
		}
		return listener, nil
	}
}

// ReverseTunnel binds a port on the remote server and redirects traffic from it to a given host and port
func (conn *SSHConn) ReverseTunnel(sourceport int, destinationhost string, destinationport int) error {

	destinationEndpoint := &Endpoint{
		Host: destinationhost,
		Port: destinationport,
	}

	listener, err := conn.ReverseTunnelListen(sourceport)
	if err != nil {
		return fmt.Errorf("%s:%d listen - %v", conn.Server.Host, sourceport, err)
	}
	defer listener.Close()

	for {
		sourceConn, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("%s:%d accept - %v", conn.Server.Host, sourceport, err)
		}
		go forward(sourceConn, destinationEndpoint)
	}
}

// CleanPrivateKey tries to fix keys that has been supplied through an environmental variable or otherwise messed up
func CleanPrivateKey(key string) string {
	re := regexp.MustCompile(`(?:-----(?:BEGIN|END) RSA PRIVATE KEY-----|\S+)`)
	lines := re.FindAllString(key, -1)
	cleanlines := []string{"-----BEGIN RSA PRIVATE KEY-----"}
	for _, line := range lines {
		if len(line) > 0 && line != "-----BEGIN RSA PRIVATE KEY-----" {
			if line == "-----END RSA PRIVATE KEY-----" {
				break
			} else {
				cleanlines = append(cleanlines, line)
			}
		}
	}
	cleanlines = append(cleanlines, "-----END RSA PRIVATE KEY-----")
	return strings.Join(cleanlines, "\n")
}
