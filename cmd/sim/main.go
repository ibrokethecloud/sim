package main

import (
	"context"
	"github.com/ibrokethecloud/sim/pkg/apiserver"
	"github.com/ibrokethecloud/sim/pkg/certs"
	"github.com/ibrokethecloud/sim/pkg/etcd"
	"github.com/ibrokethecloud/sim/pkg/kubelet"
	"github.com/ibrokethecloud/sim/pkg/objects"
	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"
	"os"
	"path/filepath"
	"time"
)

var (
	simHome    string
	bundlePath string
	resetHome  bool
	skipLoad   bool
)

func main() {

	var eg *errgroup.Group
	home, err := homedir.Dir()
	if err != nil {
		logrus.Fatalf("error querying home directory %v", err)
	}

	dir := filepath.Join(home, ".sim")

	// parsing flags
	pflag.StringVar(&simHome, "sim-home", dir, "default home directory where sim stores its configuration. default is $HOME/.sim"+
		"")

	pflag.StringVar(&bundlePath, "bundle-path", ".", "location to support bundle. default is .")
	pflag.BoolVar(&resetHome, "reset", false, "reset sim-home, will clear the contents and start a clean etcd + apiserver instance")
	pflag.BoolVar(&skipLoad, "skip-load", false, "skip load / re-load of bundle. this will ensure current etcd contents are only accessible")
	pflag.Parse()

	// clean home dir if needed
	if resetHome {
		err = os.RemoveAll(simHome)
		if err != nil {
			logrus.Fatalf("error during reset of sim-home: %v", err)
		}
	}

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	a := apiserver.APIServerConfig{}

	generatedCerts, err := certs.GenerateCerts([]string{"localhost"}, simHome)
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

	eg, egctx := errgroup.WithContext(ctx)

	k, err := kubelet.NewKubeletSimulator(egctx, generatedCerts, bundlePath)
	if err != nil {
		logrus.Fatalf("error initialisting kubelet simulator: %v", err)
	}

	eg.Go(func() error {
		return a.RunAPIServer(egctx)
	})

	eg.Go(func() error {
		return k.RunFakeKubelet()
	})

	o, err := objects.NewObjectManager(ctx, a.Config, bundlePath)
	if err != nil {
		logrus.Fatalf("error creating object manager %v", err)
	}

	err = o.WaitForNamespaces(30 * time.Second)
	if err != nil {
		logrus.Fatal(err)
	}

	if !skipLoad {
		err = o.CreateUnstructuredClusterObjects()

		if err != nil {
			logrus.Fatalf("error loading cluster scoped objects %v", err)
		}

		err = o.CreateUnstructuredObjects()
		if err != nil {
			logrus.Fatalf("error loading namespacedobjects %v", err)
		}

		logrus.Info("all resources loaded successfully")
	}

	err = eg.Wait()
	if err != nil {
		logrus.Fatalf("error from apiserver or kublet subroutine: %v", err)
	}
}
