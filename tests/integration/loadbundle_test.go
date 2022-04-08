package integration

import (
	"github.com/ibrokethecloud/sim/pkg/objects"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
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

	It("Verify Pods", func() {
		By("Verify pod objects")
		{
			Eventually(func() error {
				objs, err := objects.GenerateObjects(samplePodSpec)
				if err != nil {
					return err
				}
				for _, obj := range objs {
					unstructObj, err := o.FetchObject(obj)
					if err != nil {
						return err
					}
					logrus.Info(unstructObj.Object["status"])
				}
				return nil
			}, 5, 60).ShouldNot(HaveOccurred())

		}
	})
})
