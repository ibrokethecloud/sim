package main

import (
	"context"
	"github.com/ibrokethecloud/sim/pkg/apiserver"
	"github.com/ibrokethecloud/sim/pkg/certs"
	"github.com/ibrokethecloud/sim/pkg/etcd"
	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"path/filepath"
)

func main() {
	home, err := homedir.Dir()
	if err != nil {
		logrus.Fatalf("error querying home directory %v", err)
	}

	dir := filepath.Join(home, ".sim")
	ctx := context.Background()
	a := apiserver.APIServerConfig{}

	generatedCerts, err := certs.GenerateCerts([]string{"localhost"}, dir)
	if err != nil {
		logrus.Fatalf("error generating certificates %v", err)
	}
	a.Certs = generatedCerts

	etcdConfig, err := etcd.RunEmbeddedEtcd(ctx, filepath.Join(dir), generatedCerts)
	if err != nil {
		logrus.Fatalf("error setting up embedded etcdserver %v", err)
	}
	a.Etcd = etcdConfig

	err = a.GenerateKubeConfig(filepath.Join(dir, "admin.kubeconfig"))
	if err != nil {
		logrus.Fatalf("error generating kubeconfig %v", err)
	}

	err = a.RunAPIServer(ctx)
	if err != nil {
		logrus.Fatalf("error running API Server %v", err)
	}
}
