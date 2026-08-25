package secureadb

import (
	"context"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	esperruntime "github.com/esper-io/esper-cli/internal/runtime"
	"github.com/spf13/cobra"
)

const (
	CertificatesDirectoryEnvironment = "ESPER_CERTS_DIR"
	relayEndpointTimeout             = 160 * time.Second
	deviceCertificateTimeout         = 120 * time.Second
	localAcceptTimeout               = 5 * time.Minute
	relaySessionTimeout              = 30 * time.Minute
)

type remotePort string

func (port *remotePort) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*port = ""
		return nil
	}
	var text string
	if len(data) > 0 && data[0] == '"' {
		if err := json.Unmarshal(data, &text); err != nil {
			return err
		}
	} else {
		text = string(data)
	}
	*port = remotePort(text)
	return nil
}

func (port remotePort) number() (int, error) {
	value, err := strconv.Atoi(string(port))
	if err != nil || value < 1 || value > 65535 {
		return 0, fmt.Errorf("invalid relay client port %q", port)
	}
	return value, nil
}

type remoteADBSession struct {
	ID                string     `json:"id"`
	IP                string     `json:"ip"`
	ClientPort        remotePort `json:"client_port"`
	RemoteADBHost     string     `json:"remoteadb_host"`
	DeviceCertificate string     `json:"device_certificate"`
}

type copyResult struct {
	bytes int64
	err   error
}

func NewCommand(options *esperruntime.GlobalOptions) *cobra.Command {
	command := &cobra.Command{
		Use:   "secureadb",
		Short: "Connect ADB securely through an Esper relay",
	}
	connect := &cobra.Command{
		Use:   "connect",
		Short: "Open a pinned mutual-TLS ADB relay for a device",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			deviceID, _ := command.Flags().GetString("device")
			return runConnect(command, options, deviceID)
		},
	}
	connect.Flags().String("device", "", "device ID (defaults to the active device)")
	command.AddCommand(connect)
	return command
}

func runConnect(command *cobra.Command, options *esperruntime.GlobalOptions, deviceID string) error {
	store, err := esperruntime.NewStateStore()
	if err != nil {
		return esperruntime.NewError(esperruntime.CategoryAuth, err)
	}
	state, err := store.Load()
	if err != nil {
		return esperruntime.NewError(esperruntime.CategoryAuth, err)
	}
	if deviceID == "" && state.Active.Device != nil {
		deviceID = state.Active.Device.ID
	}
	if deviceID == "" {
		return esperruntime.NewError(esperruntime.CategoryUsage, fmt.Errorf("--device is required when no active device is set"))
	}
	credentials, err := esperruntime.ResolveCredentials(state.Config, options.Environment, options.APIKey)
	if err != nil {
		return err
	}
	if credentials.EnterpriseID == "" {
		return esperruntime.NewError(esperruntime.CategoryAuth, fmt.Errorf("enterprise ID is not configured"))
	}

	certificatesDirectory, err := defaultCertificatesDirectory()
	if err != nil {
		return esperruntime.NewError(esperruntime.CategoryAuth, err)
	}
	clientCertificatePEM, clientCertificate, deviceCertificatePath, err := prepareCertificates(certificatesDirectory, time.Now(), cryptorand.Reader)
	if err != nil {
		return fmt.Errorf("prepare secure ADB certificates: %w", err)
	}

	client := esperruntime.NewHTTPClient(credentials)
	requestPath := remoteADBCollectionPath(credentials.EnterpriseID, deviceID)
	session, err := createRemoteADBSession(command.Context(), client, requestPath, clientCertificatePEM)
	if err != nil {
		return err
	}
	detailPath := requestPath + url.PathEscape(session.ID) + "/"
	fetch := func(ctx context.Context) (remoteADBSession, error) {
		return fetchRemoteADBSession(ctx, client, detailPath)
	}

	wait := waitWithTimer
	session, err = pollRemoteADBSession(command.Context(), relayEndpointTimeout, "relay endpoint", fetch, func(session remoteADBSession) bool {
		_, portErr := session.ClientPort.number()
		return portErr == nil && (session.IP != "" || session.RemoteADBHost != "")
	}, wait)
	if err != nil {
		return err
	}
	relayHost, err := resolveRelayHost(command.Context(), session)
	if err != nil {
		return esperruntime.NewError(esperruntime.CategoryNetwork, err)
	}
	relayPort, err := session.ClientPort.number()
	if err != nil {
		return esperruntime.NewError(esperruntime.CategoryAPI, err)
	}

	session, err = pollRemoteADBSession(command.Context(), deviceCertificateTimeout, "device certificate", fetch, func(session remoteADBSession) bool {
		return session.DeviceCertificate != ""
	}, wait)
	if err != nil {
		return err
	}
	deviceCertificatePEM := []byte(session.DeviceCertificate)
	if err := os.WriteFile(deviceCertificatePath, deviceCertificatePEM, 0o600); err != nil {
		return fmt.Errorf("write device certificate: %w", err)
	}
	tlsConfig, err := pinnedTLSConfig(clientCertificate, deviceCertificatePEM)
	if err != nil {
		return esperruntime.NewError(esperruntime.CategoryAPI, err)
	}

	signalContext, stopSignals := signal.NotifyContext(command.Context(), os.Interrupt)
	defer stopSignals()
	relayConnection, err := dialTLS(signalContext, net.JoinHostPort(relayHost, strconv.Itoa(relayPort)), tlsConfig)
	if err != nil {
		return esperruntime.NewError(esperruntime.CategoryNetwork, err)
	}
	defer relayConnection.Close()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return esperruntime.NewError(esperruntime.CategoryNetwork, fmt.Errorf("listen for local ADB: %w", err))
	}
	defer listener.Close()
	if _, err := fmt.Fprintf(command.OutOrStdout(), "Secure ADB relay ready. Run:\nadb connect %s\n", listener.Addr()); err != nil {
		return err
	}

	localConnection, err := acceptOne(signalContext, listener, localAcceptTimeout)
	if err != nil {
		return esperruntime.NewError(esperruntime.CategoryNetwork, err)
	}
	defer localConnection.Close()

	started := time.Now()
	sessionContext, cancelSession := context.WithTimeout(signalContext, relaySessionTimeout)
	bytesCopied, bridgeErr := bridge(sessionContext, localConnection, relayConnection)
	cancelSession()
	if _, err := fmt.Fprintf(command.OutOrStdout(), "Session duration: %s\nBytes transferred: %d\n", time.Since(started).Round(time.Millisecond), bytesCopied); err != nil {
		return err
	}
	if bridgeErr != nil && !errors.Is(bridgeErr, context.Canceled) && !errors.Is(bridgeErr, context.DeadlineExceeded) {
		return esperruntime.NewError(esperruntime.CategoryNetwork, bridgeErr)
	}
	return nil
}

func defaultCertificatesDirectory() (string, error) {
	if directory := os.Getenv(CertificatesDirectoryEnvironment); directory != "" {
		return directory, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".esper", "certs"), nil
}

func prepareCertificates(directory string, now time.Time, random io.Reader) ([]byte, tls.Certificate, string, error) {
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, tls.Certificate{}, "", fmt.Errorf("create certificate directory: %w", err)
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return nil, tls.Certificate{}, "", fmt.Errorf("set certificate directory permissions: %w", err)
	}
	clientCertificatePath := filepath.Join(directory, "local.pem")
	clientKeyPath := filepath.Join(directory, "local.key")
	deviceCertificatePath := filepath.Join(directory, "device.pem")
	for _, path := range []string{clientCertificatePath, clientKeyPath, deviceCertificatePath} {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, tls.Certificate{}, "", fmt.Errorf("remove old certificate %s: %w", path, err)
		}
	}
	certificatePEM, keyPEM, err := generateClientCertificate(now, random)
	if err != nil {
		return nil, tls.Certificate{}, "", err
	}
	if err := os.WriteFile(clientCertificatePath, certificatePEM, 0o600); err != nil {
		return nil, tls.Certificate{}, "", fmt.Errorf("write client certificate: %w", err)
	}
	if err := os.WriteFile(clientKeyPath, keyPEM, 0o600); err != nil {
		return nil, tls.Certificate{}, "", fmt.Errorf("write client key: %w", err)
	}
	clientCertificate, err := tls.X509KeyPair(certificatePEM, keyPEM)
	if err != nil {
		return nil, tls.Certificate{}, "", fmt.Errorf("load client key pair: %w", err)
	}
	return certificatePEM, clientCertificate, deviceCertificatePath, nil
}

func generateClientCertificate(now time.Time, random io.Reader) ([]byte, []byte, error) {
	privateKey, err := rsa.GenerateKey(random, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("generate RSA client key: %w", err)
	}
	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := cryptorand.Int(random, serialLimit)
	if err != nil {
		return nil, nil, fmt.Errorf("generate certificate serial: %w", err)
	}
	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Country:      []string{"US"},
			Province:     []string{"California"},
			Locality:     []string{"Santa Clara"},
			Organization: []string{"Esper Inc."},
			CommonName:   "Esper Self-Signed Client Certificate",
		},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(random, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("create client certificate: %w", err)
	}
	certificatePEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	return certificatePEM, keyPEM, nil
}

func remoteADBCollectionPath(enterpriseID, deviceID string) string {
	return fmt.Sprintf("/v0/enterprise/%s/device/%s/remoteadb/", url.PathEscape(enterpriseID), url.PathEscape(deviceID))
}

func createRemoteADBSession(ctx context.Context, client *esperruntime.HTTPClient, path string, clientCertificate []byte) (remoteADBSession, error) {
	body, err := esperruntime.EncodeBody(map[string]string{"client_certificate": string(clientCertificate)})
	if err != nil {
		return remoteADBSession{}, err
	}
	response, err := client.Do(ctx, http.MethodPost, path, nil, body)
	if err != nil {
		return remoteADBSession{}, err
	}
	var session remoteADBSession
	if err := json.Unmarshal(response, &session); err != nil {
		return remoteADBSession{}, esperruntime.NewError(esperruntime.CategoryAPI, fmt.Errorf("decode remote ADB creation response: %w", err))
	}
	if session.ID == "" {
		return remoteADBSession{}, esperruntime.NewError(esperruntime.CategoryAPI, fmt.Errorf("remote ADB creation response has no id"))
	}
	return session, nil
}

func fetchRemoteADBSession(ctx context.Context, client *esperruntime.HTTPClient, path string) (remoteADBSession, error) {
	response, err := client.Do(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return remoteADBSession{}, err
	}
	var session remoteADBSession
	if err := json.Unmarshal(response, &session); err != nil {
		return remoteADBSession{}, esperruntime.NewError(esperruntime.CategoryAPI, fmt.Errorf("decode remote ADB status response: %w", err))
	}
	return session, nil
}

func pollRemoteADBSession(ctx context.Context, timeout time.Duration, description string, fetch func(context.Context) (remoteADBSession, error), ready func(remoteADBSession) bool, wait func(context.Context, time.Duration) error) (remoteADBSession, error) {
	pollContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	delay := time.Second
	var lastErr error
	for {
		session, err := fetch(pollContext)
		if err == nil && ready(session) {
			return session, nil
		}
		if err != nil {
			lastErr = err
		}
		if err := wait(pollContext, delay); err != nil {
			if ctx.Err() != nil {
				return remoteADBSession{}, esperruntime.NewError(esperruntime.CategoryNetwork, fmt.Errorf("wait for %s: %w", description, ctx.Err()))
			}
			message := fmt.Errorf("timed out after %s waiting for %s", timeout, description)
			if lastErr != nil {
				message = fmt.Errorf("%w (last status error: %v)", message, lastErr)
			}
			return remoteADBSession{}, esperruntime.NewError(esperruntime.CategoryNetwork, message)
		}
		if delay < 8*time.Second {
			delay *= 2
			if delay > 8*time.Second {
				delay = 8 * time.Second
			}
		}
	}
}

func waitWithTimer(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func resolveRelayHost(ctx context.Context, session remoteADBSession) (string, error) {
	if session.RemoteADBHost == "" {
		return session.IP, nil
	}
	addresses, err := net.DefaultResolver.LookupHost(ctx, session.RemoteADBHost)
	if err != nil {
		return "", fmt.Errorf("resolve remote ADB host %q: %w", session.RemoteADBHost, err)
	}
	if len(addresses) == 0 {
		return "", fmt.Errorf("resolve remote ADB host %q: no addresses returned", session.RemoteADBHost)
	}
	return addresses[0], nil
}

func pinnedTLSConfig(clientCertificate tls.Certificate, deviceCertificatePEM []byte) (*tls.Config, error) {
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(deviceCertificatePEM) {
		return nil, fmt.Errorf("device certificate is not valid PEM")
	}
	return &tls.Config{
		MinVersion:         tls.VersionTLS12,
		Certificates:       []tls.Certificate{clientCertificate},
		InsecureSkipVerify: true, // The callback below verifies the pinned certificate without a hostname check.
		VerifyPeerCertificate: func(rawCertificates [][]byte, _ [][]*x509.Certificate) error {
			if len(rawCertificates) == 0 {
				return fmt.Errorf("relay provided no certificate")
			}
			certificates := make([]*x509.Certificate, 0, len(rawCertificates))
			for _, rawCertificate := range rawCertificates {
				certificate, err := x509.ParseCertificate(rawCertificate)
				if err != nil {
					return fmt.Errorf("parse relay certificate: %w", err)
				}
				certificates = append(certificates, certificate)
			}
			intermediates := x509.NewCertPool()
			for _, certificate := range certificates[1:] {
				intermediates.AddCert(certificate)
			}
			if _, err := certificates[0].Verify(x509.VerifyOptions{
				Roots:         roots,
				Intermediates: intermediates,
				KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			}); err != nil {
				return fmt.Errorf("verify relay against device certificate: %w", err)
			}
			return nil
		},
	}, nil
}

func dialTLS(ctx context.Context, address string, config *tls.Config) (*tls.Conn, error) {
	dialer := &net.Dialer{Timeout: 30 * time.Second}
	connection, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("connect to secure ADB relay: %w", err)
	}
	secureConnection := tls.Client(connection, config)
	if err := secureConnection.HandshakeContext(ctx); err != nil {
		connection.Close()
		return nil, fmt.Errorf("perform secure ADB TLS handshake: %w", err)
	}
	return secureConnection, nil
}

func acceptOne(ctx context.Context, listener net.Listener, timeout time.Duration) (net.Conn, error) {
	if tcpListener, ok := listener.(*net.TCPListener); ok {
		if err := tcpListener.SetDeadline(time.Now().Add(timeout)); err != nil {
			return nil, fmt.Errorf("set local ADB listener deadline: %w", err)
		}
	}
	type acceptResult struct {
		connection net.Conn
		err        error
	}
	result := make(chan acceptResult, 1)
	go func() {
		connection, err := listener.Accept()
		result <- acceptResult{connection: connection, err: err}
	}()
	select {
	case <-ctx.Done():
		listener.Close()
		accepted := <-result
		if accepted.connection != nil {
			accepted.connection.Close()
		}
		return nil, fmt.Errorf("wait for local ADB client: %w", ctx.Err())
	case accepted := <-result:
		if accepted.err != nil {
			return nil, fmt.Errorf("accept local ADB client: %w", accepted.err)
		}
		return accepted.connection, nil
	}
}

func bridge(ctx context.Context, local, relay net.Conn) (int64, error) {
	results := make(chan copyResult, 2)
	copyConnection := func(destination, source net.Conn) {
		count, err := io.Copy(destination, source)
		if closeWriter, ok := destination.(interface{ CloseWrite() error }); ok {
			_ = closeWriter.CloseWrite()
		}
		results <- copyResult{bytes: count, err: err}
	}
	go copyConnection(relay, local)
	go copyConnection(local, relay)

	var bytesCopied int64
	var firstErr error
	for completed := 0; completed < 2; completed++ {
		select {
		case result := <-results:
			bytesCopied += result.bytes
			if result.err != nil && firstErr == nil && !strings.Contains(result.err.Error(), "use of closed network connection") {
				firstErr = result.err
			}
		case <-ctx.Done():
			local.Close()
			relay.Close()
			for ; completed < 2; completed++ {
				result := <-results
				bytesCopied += result.bytes
			}
			return bytesCopied, ctx.Err()
		}
	}
	return bytesCopied, firstErr
}
