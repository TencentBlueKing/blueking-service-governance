package appdefaults

import (
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appspec"
	workloaddefaults "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/workload/defaults"
)

// Platform defaults are written to every new AppModel application.
const (
	platformDefaultReplicas       int32 = 1
	platformDefaultCPURequests          = "1"
	platformDefaultCPULimits            = "2"
	platformDefaultMemoryRequests       = "2Gi"
	platformDefaultMemoryLimits         = "4Gi"
	platformDefaultMaxUnavailable       = workloaddefaults.MaxUnavailable
	platformDefaultMaxSurge             = workloaddefaults.MaxSurge
)

func newPlatformDefaultSpec(appID string) appspec.AppSpec {
	return appspec.AppSpec{
		AppID:   appID,
		EnvName: appspec.DefaultEnvName,
		Resources: &appspec.ResourcesSpec{
			Replicas:       lo.ToPtr(platformDefaultReplicas),
			CPURequests:    lo.ToPtr(platformDefaultCPURequests),
			CPULimits:      lo.ToPtr(platformDefaultCPULimits),
			MemoryRequests: lo.ToPtr(platformDefaultMemoryRequests),
			MemoryLimits:   lo.ToPtr(platformDefaultMemoryLimits),
		},
		UpdateStrategy: &appspec.UpdateStrategySpec{
			MaxUnavailable: lo.ToPtr(platformDefaultMaxUnavailable),
			MaxSurge:       lo.ToPtr(platformDefaultMaxSurge),
		},
	}
}
