package appspec

import "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/appmodelcore/appmodel"

// FromAppModel builds an app spec from an AppModel.
func FromAppModel(appID, envName string, appModel *appmodel.AppModel) *AppSpec {
	spec := &AppSpec{
		AppID:   appID,
		EnvName: envName,
	}
	for _, section := range registeredSections {
		section.fillFromAppModel(spec, appModel)
	}
	return spec
}

// ApplyToAppModel applies fields managed by appspec into the AppModel.
func ApplyToAppModel(spec *AppSpec, appModel *appmodel.AppModel) *appmodel.AppModel {
	if spec == nil {
		return appModel
	}
	for _, section := range registeredSections {
		section.applyToAppModel(spec, appModel)
	}
	return appModel
}
