package instance

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	handler "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/instance"
)

// NewPortForwardCmd returns a Command instance for 'app instance port-forward' sub command.
func NewPortForwardCmd() *cobra.Command {
	// opts is the options for port-forward
	var opts handler.PortForwardOptions

	cmd := &cobra.Command{
		Use:   "port-forward [LOCAL_PORT:]REMOTE_PORT",
		Short: "Forward a local TCP port to a single application instance Pod",
		Long: `Forward a local TCP port to a single application instance Pod.

It works like kubectl port-forward for one Pod only. To access multiple Pods,
start multiple port-forward commands with different local ports.

Port argument format:
  REMOTE_PORT         - forward the same port locally and remotely
  LOCAL_PORT:REMOTE_PORT - forward LOCAL_PORT locally to REMOTE_PORT on the Pod`,
		Example: `  # Forward localhost:8080 to pod-1:8080 (same port)
  bkms-cli app instance port-forward --app myapp --env test --instance pod-1 8080

  # Forward localhost:18080 to pod-1:8080
  bkms-cli app instance port-forward --app myapp --env test --instance pod-1 18080:8080

  # Forward with custom local address
  bkms-cli app instance port-forward --app myapp --env test --instance pod-1 18080:8080 --local-address 0.0.0.0`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			localPort, remotePort, err := handler.ParsePortArg(args[0])
			if err != nil {
				return errors.Wrap(err, "invalid port argument")
			}
			opts.LocalPort = localPort
			opts.RemotePort = remotePort
			return handler.PortForward(cmd.Context(), client.New(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.AppID, "app", "", "application ID (required)")
	cmd.Flags().StringVar(&opts.EnvName, "env", "", "environment name (required)")
	cmd.Flags().StringVar(&opts.InstanceID, "instance", "", "target Pod instance ID (required)")
	cmd.Flags().StringVar(&opts.LocalAddress, "local-address", "127.0.0.1", "local listening address")

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("env")
	_ = cmd.MarkFlagRequired("instance")

	return cmd
}
