package options

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type GRPCOptions struct {
	BindAddress          string
	MaxConcurrentStreams uint32
	MaxRecvMsgSize       int
	MaxSendMsgSize       int
	MaxConnectionIdle    time.Duration
	KeepAliveTime        time.Duration
	KeepAliveTimeout     time.Duration
	MinPingInterval      time.Duration
	PermitWithoutStream  bool
}

func NewGRPCOptions() *GRPCOptions {
	return &GRPCOptions{
		BindAddress:          "unix:///var/run/nvidia-device-api/device-api.sock",
		MaxConcurrentStreams: 250,
		MaxRecvMsgSize:       4194304,  // 4MiB
		MaxSendMsgSize:       16777216, // 16MiB
		MaxConnectionIdle:    5 * time.Minute,
		KeepAliveTime:        1 * time.Minute,
		KeepAliveTimeout:     10 * time.Second,
		MinPingInterval:      5 * time.Second,
		PermitWithoutStream:  true,
	}
}

// AddFlags adds flags related to gRPC for a specific APIServer to the specified FlagSet
func (s *GRPCOptions) AddFlags(fs *pflag.FlagSet) {
	if s == nil {
		return
	}

	fs.StringVar(&s.BindAddress, "bind-address", s.BindAddress, "The address on which to listen for gRPC requests.")
	fs.Uint32Var(&s.MaxConcurrentStreams, "max-streams-per-connection", s.MaxConcurrentStreams, "The maximum number of concurrent streams allowed per connection.")
	fs.IntVar(&s.MaxRecvMsgSize, "max-recv-msg-size", s.MaxRecvMsgSize, "The maximum message size in bytes the server can receive.")
	fs.IntVar(&s.MaxSendMsgSize, "max-send-msg-size", s.MaxSendMsgSize, "The maximum message size in bytes the server can send.")
	fs.DurationVar(&s.KeepAliveTime, "grpc-keepalive-time", s.KeepAliveTime, "Duration after which a keepalive probe is sent.")
	fs.DurationVar(&s.KeepAliveTimeout, "grpc-keepalive-timeout", s.KeepAliveTimeout, "Duration the server waits for a keepalive response.")
}

func (s *GRPCOptions) Complete() error {
	if s == nil {
		return nil
	}

	if s.BindAddress == "" {
		s.BindAddress = "unix:///var/run/nvidia-device-api/device-api.sock"
	}

	if s.MaxConcurrentStreams == 0 {
		s.MaxConcurrentStreams = 250
	}

	if s.MaxRecvMsgSize == 0 {
		s.MaxRecvMsgSize = 4194304 // 4MiB
	}

	if s.MaxSendMsgSize == 0 {
		s.MaxSendMsgSize = 16777216 // 16MiB
	}

	if s.MaxConnectionIdle == 0 {
		s.MaxConnectionIdle = 5 * time.Minute
	}

	if s.KeepAliveTime == 0 {
		s.KeepAliveTime = 1 * time.Minute
	}

	if s.KeepAliveTimeout == 0 {
		s.KeepAliveTimeout = 10 * time.Second
	}

	if s.MinPingInterval == 0 {
		s.MinPingInterval = 5 * time.Second
	}

	return nil
}

func (s *GRPCOptions) Validate() []error {
	if s == nil {
		return nil
	}

	allErrors := []error{}

	if !strings.HasPrefix(s.BindAddress, "unix://") {
		allErrors = append(allErrors, fmt.Errorf("invalid bind-address %q: must start with 'unix://'", s.BindAddress))
		return allErrors
	}
	path := strings.TrimPrefix(s.BindAddress, "unix://")
	if !filepath.IsAbs(path) {
		allErrors = append(allErrors, fmt.Errorf("invalid bind-address path %q: must be an absolute path", path))
	}
	if strings.HasSuffix(path, string(filepath.Separator)) {
		allErrors = append(allErrors, fmt.Errorf("invalid bind-address path %q: cannot end with a trailing slash", path))
	}

	if s.MaxConcurrentStreams > 10000 {
		allErrors = append(allErrors, fmt.Errorf("invalid max-streams-per-connection %d: must be no more than 10000", s.MaxConcurrentStreams))
	}
	if s.MaxRecvMsgSize > 4194304 {
		allErrors = append(allErrors, fmt.Errorf("invalid max-recv-msg-size %d: must be no more than 4MiB", s.MaxRecvMsgSize))
	}
	if s.MaxSendMsgSize > 16777216 {
		allErrors = append(allErrors, fmt.Errorf("invalid max-send-msg-size %d: must be no more than 16MiB", s.MaxRecvMsgSize))
	}

	if s.KeepAliveTime < 0 {
		allErrors = append(allErrors, fmt.Errorf("invalid grpc-keepalive-time %v: must be greater than or equal to 0", s.KeepAliveTime))
	}
	if s.KeepAliveTimeout < 0 {
		allErrors = append(allErrors, fmt.Errorf("invalid grpc-keepalive-timeout %v: must be greater than or equal to 0", s.KeepAliveTimeout))
	}
	if s.KeepAliveTimeout >= s.KeepAliveTime && s.KeepAliveTime > 0 {
		allErrors = append(allErrors, fmt.Errorf("invalid grpc-keepalive-timeout %v: must be less than grpc-keepalive-time (%v)", s.KeepAliveTimeout, s.KeepAliveTime))
	}
	if s.KeepAliveTime < s.MinPingInterval && s.KeepAliveTime > 0 {
		allErrors = append(allErrors, fmt.Errorf("invalid grpc-keepalive-time %v: must be greater than or equal to min-ping-interval (%v)", s.KeepAliveTime, s.MinPingInterval))
	}

	return allErrors
}

func (s *GRPCOptions) ApplyTo(
	bindAddress *string,
	serverOpts *[]grpc.ServerOption,
) error {
	if s == nil {
		return nil
	}

	*bindAddress = s.BindAddress

	*serverOpts = append(*serverOpts, grpc.MaxConcurrentStreams(s.MaxConcurrentStreams))
	*serverOpts = append(*serverOpts, grpc.MaxRecvMsgSize(s.MaxRecvMsgSize))
	*serverOpts = append(*serverOpts, grpc.MaxSendMsgSize(s.MaxSendMsgSize))

	*serverOpts = append(*serverOpts, grpc.KeepaliveParams(keepalive.ServerParameters{
		MaxConnectionIdle: s.MaxConnectionIdle,
		Time:              s.KeepAliveTime,
		Timeout:           s.KeepAliveTimeout,
	}))

	*serverOpts = append(*serverOpts, grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
		MinTime:             s.MinPingInterval,
		PermitWithoutStream: s.PermitWithoutStream,
	}))

	return nil
}
