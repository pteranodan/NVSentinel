package options

import (
	"context"

	cp "github.com/nvidia/nvsentinel/controlplane/apiserver/options"
	cliflag "k8s.io/component-base/cli/flag"
)

type ServerRunOptions struct {
	*cp.Options
}

type completedOptions struct {
	cp.CompletedOptions
}

type CompletedOptions struct {
	*completedOptions
}

func NewServerRunOptions() *ServerRunOptions {
	return &ServerRunOptions{
		Options: cp.NewOptions(),
	}
}

func (s *ServerRunOptions) Flags() cliflag.NamedFlagSets {
	fss := cliflag.NamedFlagSets{}

	s.Options.AddFlags(&fss)

	// TODO(user): add CLI-only flags here

	return fss
}

func (o *ServerRunOptions) Complete(ctx context.Context) (CompletedOptions, error) {
	if o == nil {
		return CompletedOptions{completedOptions: &completedOptions{}}, nil
	}

	controlplane, err := o.Options.Complete(ctx)
	if err != nil {
		return CompletedOptions{}, err
	}

	completed := completedOptions{
		CompletedOptions: controlplane,
	}

	return CompletedOptions{
		completedOptions: &completed,
	}, nil
}

func (o completedOptions) Validate() []error {
	return o.CompletedOptions.Validate()
}
