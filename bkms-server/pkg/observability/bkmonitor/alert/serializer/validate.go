package serializer

import (
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert/strategy"
)

func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		if err := v.RegisterValidation("alert_strategy_code", validateAlertStrategyCode); err != nil {
			panic("failed to register alert_strategy_code validator: " + err.Error())
		}
	}
}

func validateAlertStrategyCode(fl validator.FieldLevel) bool {
	return strategy.MonitorMetricForStrategyCode(fl.Field().String()) != ""
}
