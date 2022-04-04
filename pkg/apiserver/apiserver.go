package apiserver

import (
	"fmt"
	"github.com/ibrokethecloud/sim/pkg/certs"
	"github.com/ibrokethecloud/sim/pkg/etcd"
	"io/ioutil"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	kubeconfig "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/component-base/cli"
	"k8s.io/kubernetes/cmd/kube-apiserver/app"
	"path/filepath"
	"strings"
)

type APIServerConfig struct {
	Certs      *certs.CertInfo
	Etcd       *etcd.EtcdConfig
	KubeConfig string
	Config     *rest.Config
}

const (
	APIVersionsSupported = "v1=true,api/beta=false,api/alpha=false"
)

// Run APIServer will boot strap an API server with only core resources enabled
// No additional controllers will be scheduled
func (a *APIServerConfig) RunAPIServer() error {
	apiServer := app.NewAPIServerCommand()

	//set flag values
	apiServer.Flags().Set("tls-cert-file", a.Certs.APICert)
	apiServer.Flags().Set("tls-private-key-file", a.Certs.APICert)
	apiServer.Flags().Set("client-ca-file", a.Certs.CACert)
	apiServer.Flags().Set("service-account-key-file", a.Certs.ServiceAccountCert)
	apiServer.Flags().Set("service-account-signing-key-file", a.Certs.ServiceAccountCertKey)
	apiServer.Flags().Set("service-account-issuer", "https://localhost:6443")
	apiServer.Flags().Set("tls-cert-file", a.Certs.APICert)
	apiServer.Flags().Set("tls-private-key-file", a.Certs.APICertKey)
	apiServer.Flags().Set("runtime-config", APIVersionsSupported)
	apiServer.Flags().Set("enable-priority-and-fairness", "false")

	etcdList := strings.Join(a.Etcd.Endpoints, ",")
	apiServer.Flags().Set("etcd-servers", etcdList)

	if a.Etcd.TLS != nil {
		apiServer.Flags().Set("etcd-cafile", a.Certs.CACert)
		apiServer.Flags().Set("etcd-certfile", a.Certs.EtcdClientCert)
		apiServer.Flags().Set("etcd-keyfile", a.Certs.EtcdClientCertKey)
	}

	return cli.RunNoErrOutput(apiServer)
}

func (a *APIServerConfig) generateFlags() ([]string, error) {
	var flags []string
	if a.Certs == nil {
		return flags, fmt.Errorf("no cert info present. ensure certs have been generated")
	}

	flags = append(flags, "--cert-dir", filepath.Join(a.Certs.Dir, "kubernetes"))
	flags = append(flags, "--client-ca-file", a.Certs.CACert)
	flags = append(flags, "--service-account-key-file", a.Certs.ServiceAccountCert)
	flags = append(flags, "--service-account-signing-key-file", a.Certs.ServiceAccountCertKey)
	flags = append(flags, "--service-account-issuer", "https://localhost:6443")
	flags = append(flags, "--tls-cert-file", a.Certs.APICert)
	flags = append(flags, "--tls-private-key-file", a.Certs.APICertKey)
	flags = append(flags, "--runtime-config", APIVersionsSupported)
	flags = append(flags, "--enable-priority-and-fairness", "false")
	etcdList := strings.Join(a.Etcd.Endpoints, ",")
	flags = append(flags, "--etcd-servers", etcdList)

	if a.Etcd.TLS != nil {
		flags = append(flags, "--etcd-cafile", a.Certs.CACert)
		flags = append(flags, "--etcd-certfile", a.Certs.EtcdClientCert)
		flags = append(flags, "--etcd-certfile", a.Certs.EtcdClientCertKey)
	}
	return flags, nil
}

// APIServerConfig will generate KubeConfig to allow access to cluster
func (a *APIServerConfig) GenerateKubeConfig(path string) error {

	caCertByte, err := ioutil.ReadFile(a.Certs.CACert)
	if err != nil {
		return fmt.Errorf("error read ca cert %v", err)
	}

	adminCertByte, err := ioutil.ReadFile(a.Certs.AdminCert)
	if err != nil {
		return fmt.Errorf("error read admin cert %v", err)
	}

	adminCertKeyByte, err := ioutil.ReadFile(a.Certs.AdminCertKey)
	if err != nil {
		return fmt.Errorf("error read admin cert key %v", err)
	}

	kc := kubeconfig.NewConfig()

	cluster := kubeconfig.NewCluster()
	cluster.CertificateAuthorityData = caCertByte
	cluster.Server = "https://localhost:6443"

	authInfo := kubeconfig.NewAuthInfo()
	authInfo.ClientCertificateData = adminCertByte
	authInfo.ClientKeyData = adminCertKeyByte

	context := kubeconfig.NewContext()
	context.AuthInfo = "default"
	context.Cluster = "default"

	kc.Clusters["default"] = cluster
	kc.AuthInfos["default"] = authInfo
	kc.Contexts["default"] = context
	kc.CurrentContext = "default"

	err = clientcmd.WriteToFile(*kc, path)
	if err != nil {
		return err
	}

	clientConfig := clientcmd.NewDefaultClientConfig(*kc, nil)

	config, err := clientConfig.ClientConfig()
	if err != nil {
		return err
	}

	a.Config = config

	return nil
}
