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
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	esperruntime "github.com/esper-io/esper-cli/internal/runtime"
)

func TestGenerateClientCertificate(t *testing.T) {
	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	certificatePEM, keyPEM, err := generateClientCertificate(now, cryptorand.Reader)
	if err != nil {
		t.Fatalf("generateClientCertificate() error = %v", err)
	}
	certificateBlock, _ := pem.Decode(certificatePEM)
	if certificateBlock == nil || certificateBlock.Type != "CERTIFICATE" {
		t.Fatal("generated certificate is not PEM encoded")
	}
	certificate, err := x509.ParseCertificate(certificateBlock.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	publicKey, ok := certificate.PublicKey.(*rsa.PublicKey)
	if !ok || publicKey.N.BitLen() != 2048 {
		t.Fatalf("RSA key size = %d, want 2048", publicKey.N.BitLen())
	}
	if certificate.Subject.CommonName != "Esper Self-Signed Client Certificate" {
		t.Fatalf("common name = %q", certificate.Subject.CommonName)
	}
	if !certificate.NotAfter.Equal(now.Add(24 * time.Hour)) {
		t.Fatalf("NotAfter = %s, want %s", certificate.NotAfter, now.Add(24*time.Hour))
	}
	if err := certificate.CheckSignature(certificate.SignatureAlgorithm, certificate.RawTBSCertificate, certificate.Signature); err != nil {
		t.Fatalf("certificate is not self-signed: %v", err)
	}
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil || keyBlock.Type != "RSA PRIVATE KEY" {
		t.Fatal("generated private key is not PEM encoded")
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if privateKey.N.Cmp(publicKey.N) != 0 {
		t.Fatal("certificate and private key do not match")
	}
}

func TestPrepareCertificatesReplacesFilesWithPrivatePermissions(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "certs")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"local.pem", "local.key", "device.pem"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte("old"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	_, _, devicePath, err := prepareCertificates(directory, time.Now(), cryptorand.Reader)
	if err != nil {
		t.Fatalf("prepareCertificates() error = %v", err)
	}
	if devicePath != filepath.Join(directory, "device.pem") {
		t.Fatalf("device path = %q", devicePath)
	}
	if _, err := os.Stat(devicePath); !os.IsNotExist(err) {
		t.Fatalf("old device certificate was not removed: %v", err)
	}
	for _, name := range []string{"local.pem", "local.key"} {
		info, err := os.Stat(filepath.Join(directory, name))
		if err != nil {
			t.Fatal(err)
		}
		if permissions := info.Mode().Perm(); permissions != 0o600 {
			t.Fatalf("%s permissions = %o, want 600", name, permissions)
		}
	}
	info, err := os.Stat(directory)
	if err != nil {
		t.Fatal(err)
	}
	if permissions := info.Mode().Perm(); permissions != 0o700 {
		t.Fatalf("directory permissions = %o, want 700", permissions)
	}
}

func TestPollRemoteADBSessionStateMachine(t *testing.T) {
	responses := []remoteADBSession{
		{ID: "session-1"},
		{ID: "session-1", IP: "127.0.0.1", ClientPort: "4443"},
		{ID: "session-1", IP: "127.0.0.1", ClientPort: "4443"},
		{ID: "session-1", IP: "127.0.0.1", ClientPort: "4443", DeviceCertificate: "device-pem"},
	}
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v0/enterprise/enterprise-1/device/device-1/remoteadb/session-1/" {
			t.Errorf("request path = %q", request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer fixture-key" {
			t.Errorf("authorization = %q", request.Header.Get("Authorization"))
		}
		if requestCount >= len(responses) {
			t.Fatal("received more polling requests than expected")
		}
		_ = json.NewEncoder(writer).Encode(responses[requestCount])
		requestCount++
	}))
	defer server.Close()
	client := &esperruntime.HTTPClient{
		BaseURL: server.URL,
		APIKey:  "fixture-key",
		Client:  server.Client(),
		Retry:   esperruntime.RetryPolicy{MaxAttempts: 1},
	}
	fetch := func(ctx context.Context) (remoteADBSession, error) {
		return fetchRemoteADBSession(ctx, client, "/v0/enterprise/enterprise-1/device/device-1/remoteadb/session-1/")
	}
	var waits []time.Duration
	wait := func(_ context.Context, duration time.Duration) error {
		waits = append(waits, duration)
		return nil
	}

	endpoint, err := pollRemoteADBSession(context.Background(), time.Minute, "relay endpoint", fetch, func(session remoteADBSession) bool {
		_, portErr := session.ClientPort.number()
		return portErr == nil && session.IP != ""
	}, wait)
	if err != nil {
		t.Fatalf("poll relay endpoint: %v", err)
	}
	if endpoint.IP != "127.0.0.1" || endpoint.ClientPort != "4443" {
		t.Fatalf("endpoint = %#v", endpoint)
	}
	certificate, err := pollRemoteADBSession(context.Background(), time.Minute, "device certificate", fetch, func(session remoteADBSession) bool {
		return session.DeviceCertificate != ""
	}, wait)
	if err != nil {
		t.Fatalf("poll device certificate: %v", err)
	}
	if certificate.DeviceCertificate != "device-pem" {
		t.Fatalf("device certificate = %q", certificate.DeviceCertificate)
	}
	if requestCount != 4 {
		t.Fatalf("request count = %d, want 4", requestCount)
	}
	if len(waits) != 2 || waits[0] != time.Second || waits[1] != time.Second {
		t.Fatalf("waits = %v, want [1s 1s]", waits)
	}
}

func TestPollRemoteADBSessionTimeout(t *testing.T) {
	wait := func(ctx context.Context, _ time.Duration) error {
		<-ctx.Done()
		return ctx.Err()
	}
	_, err := pollRemoteADBSession(context.Background(), time.Millisecond, "relay endpoint", func(context.Context) (remoteADBSession, error) {
		return remoteADBSession{}, nil
	}, func(remoteADBSession) bool { return false }, wait)
	if err == nil {
		t.Fatal("pollRemoteADBSession() error = nil")
	}
	if category := esperruntime.ErrorCategory(err); category != esperruntime.CategoryNetwork {
		t.Fatalf("error category = %q, want network", category)
	}
}

func TestPollRemoteADBSessionParentCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	wait := func(context.Context, time.Duration) error {
		cancel()
		return ctx.Err()
	}
	_, err := pollRemoteADBSession(ctx, time.Minute, "relay endpoint", func(context.Context) (remoteADBSession, error) {
		return remoteADBSession{}, nil
	}, func(remoteADBSession) bool { return false }, wait)
	if err == nil || err.Error() != "wait for relay endpoint: context canceled" {
		t.Fatalf("pollRemoteADBSession() error = %v", err)
	}
}

func TestPinnedTLSConfig(t *testing.T) {
	clientCertificatePEM, clientKeyPEM, err := generateClientCertificate(time.Now(), cryptorand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	clientCertificate, err := tls.X509KeyPair(clientCertificatePEM, clientKeyPEM)
	if err != nil {
		t.Fatal(err)
	}
	serverCertificate, serverCertificatePEM := newServerCertificate(t, "relay.invalid")
	_, unrelatedCertificatePEM := newServerCertificate(t, "other.invalid")

	t.Run("accepts pinned certificate without hostname verification", func(t *testing.T) {
		config, err := pinnedTLSConfig(clientCertificate, serverCertificatePEM)
		if err != nil {
			t.Fatal(err)
		}
		if err := localTLSHandshake(config, serverCertificate); err != nil {
			t.Fatalf("TLS handshake error = %v", err)
		}
	})

	t.Run("rejects certificate outside pin", func(t *testing.T) {
		config, err := pinnedTLSConfig(clientCertificate, unrelatedCertificatePEM)
		if err != nil {
			t.Fatal(err)
		}
		if err := localTLSHandshake(config, serverCertificate); err == nil {
			t.Fatal("TLS handshake unexpectedly trusted an unpinned certificate")
		}
	})
}

func TestBridgeStopsOnContextCancellation(t *testing.T) {
	localBridge, localClient := net.Pipe()
	relayBridge, relayServer := net.Pipe()
	defer localClient.Close()
	defer relayServer.Close()
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := bridge(ctx, localBridge, relayBridge)
		result <- err
	}()

	written := make(chan error, 1)
	go func() {
		_, err := localClient.Write([]byte("adb"))
		written <- err
	}()
	buffer := make([]byte, 3)
	if _, err := relayServer.Read(buffer); err != nil {
		t.Fatal(err)
	}
	if err := <-written; err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case err := <-result:
		if err != context.Canceled {
			t.Fatalf("bridge() error = %v, want context canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("bridge did not stop after cancellation")
	}
}

func newServerCertificate(t *testing.T, commonName string) (tls.Certificate, []byte) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(cryptorand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: commonName},
		DNSNames:              []string{commonName},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(cryptorand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	certificatePEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	certificate, err := tls.X509KeyPair(certificatePEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	return certificate, certificatePEM
}

func localTLSHandshake(clientConfig *tls.Config, serverCertificate tls.Certificate) error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer listener.Close()
	serverResult := make(chan error, 1)
	go func() {
		connection, acceptErr := listener.Accept()
		if acceptErr != nil {
			serverResult <- acceptErr
			return
		}
		defer connection.Close()
		secureConnection := tls.Server(connection, &tls.Config{
			Certificates: []tls.Certificate{serverCertificate},
			ClientAuth:   tls.RequireAnyClientCert,
			MinVersion:   tls.VersionTLS12,
		})
		if err := secureConnection.Handshake(); err != nil {
			serverResult <- err
			return
		}
		if len(secureConnection.ConnectionState().PeerCertificates) != 1 {
			serverResult <- fmt.Errorf("client certificate was not presented")
			return
		}
		serverResult <- nil
	}()

	connection, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	if err != nil {
		return err
	}
	secureConnection := tls.Client(connection, clientConfig)
	clientErr := secureConnection.Handshake()
	secureConnection.Close()
	serverErr := <-serverResult
	if clientErr != nil {
		return clientErr
	}
	return serverErr
}
