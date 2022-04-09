package controllers

import (
	"context"
	"github.com/ibrokethecloud/sim/pkg/controllers/node"
	"github.com/rancher/lasso/pkg/cache"
	"github.com/rancher/lasso/pkg/client"
	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/wrangler/pkg/generated/controllers/core"
	"github.com/rancher/wrangler/pkg/start"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/workqueue"
	"time"
)

func Start(ctx context.Context, cfg *rest.Config) error {
	rateLimit := workqueue.NewItemExponentialFailureRateLimiter(5*time.Millisecond, 5*time.Minute)
	workqueue.DefaultControllerRateLimiter()
	clientFactory, err := client.NewSharedClientFactory(cfg, nil)
	if err != nil {
		return err
	}

	cacheFactory := cache.NewSharedCachedFactory(clientFactory, nil)
	scf := controller.NewSharedControllerFactory(cacheFactory, &controller.SharedControllerFactoryOptions{
		DefaultRateLimiter: rateLimit,
		DefaultWorkers:     5,
	})

	if err != nil {
		return err
	}

	coreControllers, err := core.NewFactoryFromConfigWithOptions(cfg, &core.FactoryOptions{
		SharedControllerFactory: scf,
	})
	if err != nil {
		return err
	}

	node.Register(ctx, coreControllers.Core().V1().Node())
	return start.All(ctx, 1, coreControllers)
}
