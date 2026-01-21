package app

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/cli/globalflag"
	"k8s.io/component-base/logs"
	logsapi "k8s.io/component-base/logs/api/v1"
	"k8s.io/component-base/term"
	utilversion "k8s.io/component-base/version"
	"k8s.io/component-base/version/verflag"
	"k8s.io/klog/v2"

	"github.com/nvidia/nvsentinel/cmd/device-apiserver/app/options"
)

func init() {
	// TODO(dhuenecke): add required global registrations
}

// NewAPIServerCommand creates a *cobra.Command object with default parameters
func NewAPIServerCommand() *cobra.Command {
	s := options.NewServerRunOptions()

	cmd := &cobra.Command{
		Use: "device-apiserver",
		Long: `The Device API server validates and configures data
for the api objects which include gpus and others. The API Server services
gRPC operations and provides the frontend to a node's shared state through
which all other node-local components interact.`,

		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			verflag.PrintAndExitIfRequested()
			fs := cmd.Flags()

			logsapi.ReapplyHandling = logsapi.ReapplyHandlingIgnoreUnchanged

			// Activate logging as soon as possible, after that
			// show flags with the final logging configuration.
			if err := logsapi.ValidateAndApply(s.Logs, nil); err != nil {
				return err
			}
			cliflag.PrintFlags(fs)

			// set default options
			completedOptions, err := s.Complete(ctx)
			if err != nil {
				return err
			}

			// validate options
			if errs := completedOptions.Validate(); len(errs) != 0 {
				return utilerrors.NewAggregate(errs)
			}

			// TODO(dhuenecke): add component version metrics
			return Run(ctx, completedOptions)
		},
		Args: func(cmd *cobra.Command, args []string) error {
			for _, arg := range args {
				if len(arg) > 0 {
					return fmt.Errorf("%q does not take any arguments, got %q", cmd.CommandPath(), args)
				}
			}
			return nil
		},
	}

	fs := cmd.Flags()
	namedFlagSets := s.Flags()
	// TODO(dhuenecke): add flagz functionality (introspection into whats configuration, flags, the server is running with)
	verflag.AddFlags(namedFlagSets.FlagSet("global"))
	globalflag.AddGlobalFlags(namedFlagSets.FlagSet("global"), cmd.Name(), logs.SkipLoggingConfigurationFlags())
	for _, f := range namedFlagSets.FlagSets {
		fs.AddFlagSet(f)
	}

	cols, _, _ := term.TerminalSize(cmd.OutOrStdout())
	cliflag.SetUsageAndHelpFunc(cmd, namedFlagSets, cols)

	return cmd
}

// Run runs the specified APIServer. This should never exit.
func Run(ctx context.Context, opts options.CompletedOptions) error {
	logger := klog.FromContext(ctx).WithValues("node", opts.NodeName)
	ctx = klog.NewContext(ctx, logger)

	logger.Info("Initializing Device API Server", "version", utilversion.Get())
	logger.V(2).Info("Golang settings", "GOGC", os.Getenv("GOGC"), "GOMAXPROCS", os.Getenv("GOMAXPROCS"), "GOTRACEBACK", os.Getenv("GOTRACEBACK"))

	config, err := NewConfig(ctx, opts)
	if err != nil {
		return err
	}

	completed, err := config.Complete()
	if err != nil {
		return err
	}

	server, err := completed.APIs.New()
	if err != nil {
		return err
	}

	prepared := server.PrepareRun()

	return prepared.Run(ctx)
}
