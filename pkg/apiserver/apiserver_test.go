package apiserver

import (
	"context"
	"github.com/ibrokethecloud/sim/pkg/certs"
	"github.com/ibrokethecloud/sim/pkg/etcd"
	"io/ioutil"
	"path/filepath"
	"testing"
	"time"
)

func TestRunAPIServer(t *testing.T) {

	dir, err := ioutil.TempDir("/tmp", "apiserver-")
	if err != nil {
		t.Fatalf("error setting up temp directory for apiserver %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	a := APIServerConfig{}

	generatedCerts, err := certs.GenerateCerts([]string{"localhost"}, dir)
	if err != nil {
		t.Fatalf("error generating certificates for sim %v", err)
	}
	a.Certs = generatedCerts

	etcdConfig, err := etcd.RunEmbeddedEtcd(ctx, filepath.Join(dir), generatedCerts)
	if err != nil {
		t.Fatalf("error setting up embedded etcdserver")
	}
	a.Etcd = etcdConfig

	err = a.GenerateKubeConfig(filepath.Join(dir, "admin.kubeconfig"))
	if err != nil {
		t.Fatalf("error generating kubeconfig %v", err)
	}

	err = a.RunAPIServer()
	if err != nil {
		t.Fatalf("error running API Server")
	}
}
