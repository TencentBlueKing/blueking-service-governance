package deploy

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/handler/deploy"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
)

// NewUpdateCmd returns a Command instance for 'app deploy update' sub command
// 更新模式：
// Full update：全量更新；更新内容：镜像Tag+配置；该更新模式会重建Workload、Pod。
// Config update：全量更新；更新内容：仅配置；该更新模式会重建Workload、Pod。
// Image update： 全量更新；更新内容：仅镜像Tag + 更新策略，策略：RollingUpdate（滚动更新，重建 Pod）、InplaceUpdate（原地更新，重启 Pod）
// Grayscale update： 灰度更新，更新内容：镜像Tag + 实例名称，默认为 InplaceUpdate（原地更新，重启 Pod）
func NewUpdateCmd() *cobra.Command {
	var appID, envName, updateSpecFile, workspaceID string

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update an existing application deployment",
		Long: `Update an existing deployment for an application.

The --env flag supports multiple environment names separated by commas (e.g. --env prod,staging).
When multiple environments are specified, the update will be executed for each environment sequentially.
Environment names are validated against the workspace before updating.

If you have set a default workspace using 'workspace set', the --workspace flag
is optional. Otherwise, you must specify it explicitly.

Supported application types: trpc, taf.

This command supports four update modes, specified via the 'updateMode' field in the YAML spec file:

  [Full] - Full update with image and configuration
    imageTag:      Image tag (required)
    Rebuilds Workload and Pod.

  [Config] - Configuration only update
    No additional fields required.
    Rebuilds Workload and Pod.

  [Image] - Image only update with strategy
    imageTag:      Image tag (required)
    strategy:  Update strategy (required): InplaceUpdate or RollingUpdate
    - InplaceUpdate: In-place update, restarts Pod (recommended)
    - RollingUpdate: Rolling update, rebuilds Pod

  [Grayscale] - Grayscale/Canary update for specific instances
    imageTag:      Image tag (required)
    instanceIDs: Pod names separated by semicolon (required)
    Uses InplaceUpdate strategy by default.`,
		Example: `  # 1. Full update (update-full.yaml):
  updateMode: Full
  imageTag: v1.0.0

  # Execute full update
  bkms-cli app deploy update --app my-app --env prod -f update-full.yaml

  # 2. Config update (update-config.yaml):
  updateMode: Config

  # Execute config update
  bkms-cli app deploy update --app my-app --env prod -f update-config.yaml

  # 3. Image update (update-image.yaml):
  updateMode: Image
  imageTag: v1.0.0
  strategy: InplaceUpdate

  # Execute image update
  bkms-cli app deploy update --app my-app --env prod -f update-image.yaml

  # 4. Grayscale update (update-grayscale.yaml):
  updateMode: Grayscale
  imageTag: v1.0.0
  instanceIDs: "pod-1;pod-2"
  # Or use array format:
  # instanceIDs:
  #   - pod-1
  #   - pod-2
  #   - pod-3

  # Execute grayscale update
  bkms-cli app deploy update --app my-app --env prod -f update-grayscale.yaml

  # Specify workspace explicitly
  bkms-cli app deploy update --workspace ws-demo --app my-app --env prod -f update-full.yaml

  # 5. Update multiple environments at once
  bkms-cli app deploy update --app my-app --env prod,staging,test -f update-image.yaml`,
		PreRun: cmdutil.CommonPreRun,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceID = cmdutil.GetWorkspaceID(workspaceID)
			if err := deploy.UpdateDeploy(cmd.Context(), workspaceID, appID, envName, updateSpecFile); err != nil {
				return errors.Wrap(err, "update app deploy")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&workspaceID, "workspace", "", "workspace id")
	cmd.Flags().StringVar(&appID, "app", "", "application ID")
	cmd.Flags().StringVar(&envName, "env", "", "environment name")
	cmd.Flags().StringVarP(&updateSpecFile, "update-spec-file", "f", "", "update spec file path")

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("env")
	_ = cmd.MarkFlagRequired("update-spec-file")

	return cmd
}
