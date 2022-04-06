package integration

import (
	"context"
	"github.com/ibrokethecloud/sim/pkg/apiserver"
	"github.com/ibrokethecloud/sim/pkg/certs"
	"github.com/ibrokethecloud/sim/pkg/etcd"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	_ "github.com/rancher/wrangler/pkg/generated/controllers/apiextensions.k8s.io/v1"
	_ "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	"golang.org/x/sync/errgroup"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	timeout      = time.Second * 10
	duration     = time.Second * 10
	setupTimeout = 300
	samplesPath  = "../../samples/sampleSupportBundle"
)

func TestSim(t *testing.T) {
	defer GinkgoRecover()
	RegisterFailHandler(Fail)
	RunSpecs(t, "sim integration")
}

var (
	ctx    context.Context
	cancel context.CancelFunc
	a      apiserver.APIServerConfig
	dir    string
	eg     *errgroup.Group
	egctx  context.Context
)

var _ = BeforeSuite(func(done Done) {
	defer close(done)
	var err error
	By("starting test cluster")
	ctx, cancel = context.WithCancel(context.TODO())

	dir, err = ioutil.TempDir("/tmp", "integration-")
	Expect(err).ToNot(HaveOccurred())

	certificates, err := certs.GenerateCerts([]string{"localhost", "127.0.0.1"},
		dir)
	Expect(err).ToNot(HaveOccurred())
	a.Certs = certificates

	etcd, err := etcd.RunEmbeddedEtcd(ctx, dir, certificates)
	Expect(err).ToNot(HaveOccurred())
	a.Etcd = etcd

	Expect(a.GenerateKubeConfig(filepath.Join(dir, "admin.kubeconfig"))).ToNot(HaveOccurred())

	eg, egctx = errgroup.WithContext(ctx)
	Expect(err).ToNot(HaveOccurred())

	eg.Go(func() error {
		return a.RunAPIServer(egctx)
	})

	eg.Go(func() error {
		return eg.Wait()
	})

	// wait for apiserver to start
	time.Sleep(30 * time.Second)
}, setupTimeout)

var _ = AfterSuite(func(done Done) {
	//time.Sleep(300 * time.Second)
	defer os.Remove(dir)
	defer close(done)
	cancel()
}, setupTimeout)
