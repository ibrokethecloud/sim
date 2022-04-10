package kubelet

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/ibrokethecloud/sim/pkg/certs"
	"github.com/ibrokethecloud/sim/pkg/util"
	"github.com/virtual-kubelet/node-cli/opts"
	"github.com/virtual-kubelet/virtual-kubelet/node/api"
	"io"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	"log"
	"net/http"
	"path/filepath"
	"time"
)

const (
	defaultStreamIdleTimeout     = 10 * time.Second
	defaultStreamCreationTimeout = 10 * time.Second
)

type KubeletSimulator struct {
	ctx        context.Context
	certs      *certs.CertInfo
	bundlePath string
}

func NewKubeletSimulator(ctx context.Context, certs *certs.CertInfo, bundlePath string) *KubeletSimulator {
	// check bundle path exists
	return &KubeletSimulator{
		ctx:        ctx,
		certs:      certs,
		bundlePath: bundlePath,
	}
}
func (k *KubeletSimulator) RunFakeKubelet() error {
	routes := http.NewServeMux()
	podRoutes := api.PodHandlerConfig{
		RunInContainer:        k.runInContainer,
		GetContainerLogs:      k.getContainerLogs,
		GetPods:               k.getPods,
		StreamIdleTimeout:     defaultStreamIdleTimeout,
		StreamCreationTimeout: opts.DefaultStreamCreationTimeout,
	}

	api.AttachPodRoutes(podRoutes, routes, false)

	tlsConfig, err := loadTLSConfig(k.ctx, k.certs.KubeletCert, k.certs.KubeletCertKey, k.certs.CACert, true, false)
	if err != nil {
		return err

	}
	s := &http.Server{
		Handler:   routes,
		TLSConfig: tlsConfig,
	}

	l, err := tls.Listen("tcp", "127.0.0.1:10250", tlsConfig)
	if err != nil {
		return err
	}

	go func() {
		<-k.ctx.Done()
		s.Shutdown(k.ctx)
	}()

	return s.Serve(l)

}

// runInContainer does nothing
func (k *KubeletSimulator) runInContainer(ctx context.Context, namespace, name, container string, cmd []string, attach api.AttachIO) error {
	return nil
}

// getPods returns pod information
func (k *KubeletSimulator) getPods(ctx context.Context) ([]*corev1.Pod, error) {
	log.Printf("GetPods from path %s\n", k.bundlePath)
	return util.GeneratePodList(k.bundlePath)
}

// getContainerLogs streams the logs from the bundle
func (k *KubeletSimulator) getContainerLogs(ctx context.Context, namespace, podName, containerName string, opts api.ContainerLogOpts) (io.ReadCloser, error) {
	contents, err := readLogFiles(k.bundlePath, namespace, podName, containerName)
	if err != nil {
		return nil, err
	}
	return ioutil.NopCloser(contents), nil
}

func readLogFiles(path, namespace, name, container string) (io.Reader, error) {
	abs, err := filepath.Abs(filepath.Join(path, "logs", namespace, name, container+".log"))
	if err != nil {
		return nil, err
	}

	content, err := ioutil.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(content), nil
}

func loadTLSConfig(ctx context.Context, certPath, keyPath, caPath string, allowUnauthenticatedClients, authWebhookEnabled bool) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("error loading tls certs %v", err)
	}

	var (
		AcceptedCiphers = []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,

			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		}
		caPool     *x509.CertPool
		clientAuth = tls.RequireAndVerifyClientCert
	)

	if allowUnauthenticatedClients {
		clientAuth = tls.NoClientCert
	}
	if authWebhookEnabled {
		clientAuth = tls.RequestClientCert
	}

	if caPath != "" {
		caPool = x509.NewCertPool()
		pem, err := ioutil.ReadFile(caPath)
		if err != nil {
			return nil, err
		}
		if !caPool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("error appending ca cert to certificate pool")
		}
	}

	return &tls.Config{
		Certificates:             []tls.Certificate{cert},
		MinVersion:               tls.VersionTLS12,
		PreferServerCipherSuites: true,
		CipherSuites:             AcceptedCiphers,
		ClientCAs:                caPool,
		ClientAuth:               clientAuth,
	}, nil
}
