package appdefaults

import (
	"go.uber.org/fx"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

// FxModule provides the application-default rule store.
var FxModule = fx.Module("appdefaults",
	database.PrivateFxModule,
	fx.Provide(
		fx.Annotate(NewRuleStoreMongo, fx.As(new(RuleStore))),
	),
)
