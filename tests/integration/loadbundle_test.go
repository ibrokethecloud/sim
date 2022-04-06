package integration

import (
	"github.com/ibrokethecloud/sim/pkg/objects"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Process Support Bundle", func() {
	var o *objects.ObjectManager
	var err error
	BeforeEach(func() {
		Eventually(func() error {
			o, err = objects.NewObjectManager(ctx, a.Config, samplesPath)
			return err
		}, 5, 60).ShouldNot(HaveOccurred())
	})

	It("Load cluster scoped objects", func() {
		Eventually(func() error {
			return o.CreateUnstructuredClusterObjects()
		}, 5, 60).ShouldNot(HaveOccurred())
	})

	It("Load namespace scoped objects", func() {
		Eventually(func() error {
			return o.CreateUnstructuredObjects()
		}, 5, 60).ShouldNot(HaveOccurred())
	})
})
