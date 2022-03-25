package main

import (
	"context"
	supportbundle "github.com/ibrokethecloud/sim/pkg/provider"
	"github.com/ibrokethecloud/sim/pkg/setup"
	"github.com/spf13/pflag"
	"github.com/virtual-kubelet/node-cli/opts"
	"k8s.io/client-go/tools/clientcmd"
	"strings"

	"github.com/sirupsen/logrus"
	cli "github.com/virtual-kubelet/node-cli"
	logruscli "github.com/virtual-kubelet/node-cli/logrus"
	"github.com/virtual-kubelet/node-cli/provider"
	"github.com/virtual-kubelet/virtual-kubelet/log"
	logruslogger "github.com/virtual-kubelet/virtual-kubelet/log/logrus"
)

var (
	buildVersion    = "N/A"
	buildTime       = "N/A"
	k8sVersion      = "v1.19.10" // This should follow the version of k8s.io/client-go we are importing
	numberOfWorkers = 1
)

func main() {
	ctx := cli.ContextWithCancelOnSignal(context.Background())
	logger := logrus.StandardLogger()

	log.L = logruslogger.FromLogrus(logrus.NewEntry(logger))
	logConfig := &logruscli.Config{LogLevel: "info"}

	o, err := opts.FromEnv()
	if err != nil {
		log.G(ctx).Fatal(err)
	}
	o.Provider = "supportbundle"
	o.Version = strings.Join([]string{k8sVersion, "vk-supportbundle", buildVersion}, "-")
	o.PodSyncWorkers = numberOfWorkers
	// additional flags
	var path, internalAddress string
	supportBundleFlags := pflag.NewFlagSet("supportBundleFlags", pflag.ExitOnError)
	supportBundleFlags.StringVar(&path, "path", ".", "path to support bundle")
	supportBundleFlags.StringVar(&internalAddress, "internal-address", "127.0.0.1", "internal address advertised by kubelet")

	node, err := cli.New(ctx,
		cli.WithBaseOpts(o),
		cli.WithProvider("supportbundle", func(cfg provider.InitConfig) (provider.Provider, error) {
			return supportbundle.NewProvider(cfg.ResourceManager, cfg.NodeName, path, internalAddress)
		}),
		// Adds flags and parsing for using logrus as the configured logger
		cli.WithPersistentFlags(logConfig.FlagSet()),
		cli.WithPersistentFlags(supportBundleFlags),
		cli.WithPersistentPreRunCallback(func() error {
			err = logruscli.Configure(logConfig, logger)
			if err != nil {
				return err
			}

			config, err := clientcmd.BuildConfigFromFlags("", o.KubeConfigPath)
			if err != nil {
				return err
			}

			return setup.SetupPodObjects(ctx, config, path)
		}),
	)

	if err != nil {
		panic(err)
	}
	// Args can be specified here, or os.Args[1:] will be used.
	if err := node.Run(ctx); err != nil {
		panic(err)
	}
}
