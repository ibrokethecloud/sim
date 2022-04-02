package apiserver

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/ibrokethecloud/sim/pkg/certs"
	"github.com/ibrokethecloud/sim/pkg/etcd"
	"io/fs"
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
	flags, err := a.generateFlags()
	if err != nil {
		return err
	}

	apiServer := app.NewAPIServerCommand()
	apiServer.SetArgs(flags)

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
	caCert := base64.StdEncoding.EncodeToString(caCertByte)

	adminCertByte, err := ioutil.ReadFile(a.Certs.AdminCert)
	if err != nil {
		return fmt.Errorf("error read admin cert %v", err)
	}
	adminCert := base64.StdEncoding.EncodeToString(adminCertByte)

	adminCertKeyBte, err := ioutil.ReadFile(a.Certs.AdminCertKey)
	if err != nil {
		return fmt.Errorf("error read admin cert key %v", err)
	}
	adminCertKey := base64.StdEncoding.EncodeToString(adminCertKeyBte)

	kc := &kubeconfig.Config{}
	kc.Kind = "Cluster"
	kc.APIVersion = "v1"
	kc.Contexts = map[string]*kubeconfig.Context{
		"default": &kubeconfig.Context{
			Cluster:   "sim",
			AuthInfo:  "admin",
			Namespace: "default",
		},
	}
	kc.CurrentContext = "default"
	kc.AuthInfos = map[string]*kubeconfig.AuthInfo{
		"sim": &kubeconfig.AuthInfo{
			ClientCertificate: adminCert,
			ClientKey:         adminCertKey,
		},
	}

	kc.Clusters = map[string]*kubeconfig.Cluster{
		"sim": &kubeconfig.Cluster{
			Server:               "https://localhost:6443",
			CertificateAuthority: caCert,
		},
	}

	kcBytes, err := json.Marshal(kc)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path, kcBytes, fs.FileMode(0644))
	if err != nil {
		return err
	}

	a.KubeConfig = path

	clientConfig, err := clientcmd.NewClientConfigFromBytes(kcBytes)
	if err != nil {
		return err
	}

	config, err := clientConfig.ClientConfig()
	if err != nil {
		return err
	}

	a.Config = config

	return nil
}
