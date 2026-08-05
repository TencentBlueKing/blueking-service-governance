package serializer

import (
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"

	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
	alertstrategy "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/bkmonitor/alert/strategy"
)

var _ = Describe("Alert serializer", func() {
	Describe("NewAlertEventOutput", func() {
		It("keeps assignees from alert event", func() {
			output := NewAlertEventOutput(bkmapi.AlertEvent{
				ID:         "alert-1",
				AlertName:  "cpu high",
				Status:     "ABNORMAL",
				Severity:   2,
				StrategyID: 1001,
				Assignee:   []string{"alice", "bob"},
				BeginTime:  1710000000,
				EndTime:    1710000300,
				LatestTime: 1710000300,
				CreateTime: 1710000000,
			}, "strategy-1")

			Expect(output.Assignee).To(Equal([]string{"alice", "bob"}))
		})
	})

	Describe("AlertStrategyOutput", func() {
		It("keeps app fields from model", func() {
			envID := bson.NewObjectID()
			output := new(AlertStrategyOutput).FromModel(alertstrategy.AlertStrategy{
				ID:           bson.NewObjectID(),
				WorkspaceID:  "ws-1",
				AppID:        "app-1",
				AppName:      "demo-app",
				StrategyCode: "cpu_limit_usage_high",
				EffectiveTimeRange: alertstrategy.EffectiveTimeRange{
					StartTime: "09:00:00",
					EndTime:   "18:00:00",
				},
				EffectiveScope: alertstrategy.EffectiveScope{
					Type:   alertstrategy.EffectiveScopeSpecificEnvs,
					EnvIDs: []bson.ObjectID{envID},
				},
				RemoteRefs: []alertstrategy.RemoteStrategyRef{{
					EnvID:              envID,
					EnvName:            "prod",
					TrafficLaneName:    "feature-a",
					RemoteStrategyName: "demo",
					RemoteStrategyID:   1001,
				}},
			})

			Expect(output.AppID).To(Equal("app-1"))
			Expect(output.AppName).To(Equal("demo-app"))
			Expect(output.EffectiveTimeRange.StartTime).To(Equal("09:00:00"))
			Expect(output.EffectiveTimeRange.EndTime).To(Equal("18:00:00"))
			Expect(output.EffectiveScope.EnvIDs).To(Equal([]string{envID.Hex()}))
			Expect(output.RemoteRefs).To(HaveLen(1))
			Expect(output.RemoteRefs[0].TrafficLaneName).To(Equal("feature-a"))
			Expect(output.RemoteRefs[0].RemoteStrategyName).To(Equal("demo"))
		})
	})

	Describe("CreateAlertStrategyBody", func() {
		It("converts request body to create req", func() {
			envID := bson.NewObjectID()
			body := CreateAlertStrategyBody{
				StrategyCode:  "cpu_limit_usage_high",
				DisplayName:   "CPU Limit",
				MonitorMetric: "container_cpu_usage_seconds_total",
				Severity:      1,
				Threshold: ThresholdConfigInput{
					Method: "gte",
					Value:  90,
				},
				TriggerCondition: TriggerConditionInput{
					Count:       3,
					CheckWindow: 5,
				},
				RecoverCondition: RecoverConditionInput{
					CheckWindow: 10,
				},
				EffectiveTimeRange: EffectiveTimeRangeInput{
					StartTime: "09:00:00",
					EndTime:   "18:00:00",
				},
				EffectiveScope: EffectiveScopeInput{
					Type:   "specific_envs",
					EnvIDs: []string{envID.Hex()},
				},
				NoticeGroupIDs: []int64{1001, 1002},
				Enabled:        true,
			}

			req, err := body.ToCreateReq("ws-1", "app-1", "demo-app", "tester")

			Expect(err).NotTo(HaveOccurred())
			Expect(req).To(Equal(&alertstrategy.CreateReq{
				WorkspaceID:   "ws-1",
				AppID:         "app-1",
				AppName:       "demo-app",
				StrategyCode:  "cpu_limit_usage_high",
				DisplayName:   "CPU Limit",
				MonitorMetric: "container_cpu_usage_seconds_total",
				Severity:      alertstrategy.AlertSeverityFatal,
				Threshold: alertstrategy.ThresholdConfig{
					Method: "gte",
					Value:  90,
				},
				TriggerCondition: alertstrategy.TriggerCondition{
					Count:       3,
					CheckWindow: 5,
				},
				RecoverCondition: alertstrategy.RecoverCondition{
					CheckWindow: 10,
				},
				EffectiveTimeRange: alertstrategy.EffectiveTimeRange{
					StartTime: "09:00:00",
					EndTime:   "18:00:00",
				},
				EffectiveScope: alertstrategy.EffectiveScope{
					Type:   alertstrategy.EffectiveScopeSpecificEnvs,
					EnvIDs: []bson.ObjectID{envID},
				},
				NoticeGroupIDs: []int64{1001, 1002},
				Enabled:        true,
				Operator:       "tester",
			}))
		})

		It("returns error for invalid env id", func() {
			_, err := CreateAlertStrategyBody{
				EffectiveScope: EffectiveScopeInput{
					Type:   "specific_envs",
					EnvIDs: []string{"bad-id"},
				},
			}.ToCreateReq("ws-1", "app-1", "demo-app", "tester")

			Expect(err).To(HaveOccurred())
		})
	})

	Describe("UpdateAlertStrategyBody", func() {
		It("converts request body to update req", func() {
			envID := bson.NewObjectID()
			displayName := "After Update"
			severity := 2
			enabled := true
			body := UpdateAlertStrategyBody{
				DisplayName: &displayName,
				Severity:    &severity,
				Threshold: &ThresholdConfigInput{
					Method: "lt",
					Value:  50,
				},
				TriggerCondition: &TriggerConditionInput{
					Count:       2,
					CheckWindow: 3,
				},
				RecoverCondition: &RecoverConditionInput{
					CheckWindow: 8,
				},
				EffectiveTimeRange: &EffectiveTimeRangeInput{
					StartTime: "08:00:00",
					EndTime:   "20:00:00",
				},
				EffectiveScope: &EffectiveScopeInput{
					Type:     "env_type",
					EnvTypes: []string{"production"},
					EnvIDs:   []string{envID.Hex()},
				},
				NoticeGroupIDs: []int64{2001},
				Enabled:        &enabled,
			}

			req, err := body.ToUpdateReq("tester")

			Expect(err).NotTo(HaveOccurred())
			Expect(req.DisplayName).NotTo(BeNil())
			Expect(*req.DisplayName).To(Equal("After Update"))
			Expect(req.Severity).NotTo(BeNil())
			Expect(*req.Severity).To(Equal(alertstrategy.AlertSeverityWarning))
			Expect(req.Threshold).To(Equal(&alertstrategy.ThresholdConfig{Method: "lt", Value: 50}))
			Expect(req.TriggerCondition).To(Equal(&alertstrategy.TriggerCondition{Count: 2, CheckWindow: 3}))
			Expect(req.RecoverCondition).To(Equal(&alertstrategy.RecoverCondition{CheckWindow: 8}))
			Expect(req.EffectiveTimeRange).To(Equal(&alertstrategy.EffectiveTimeRange{
				StartTime: "08:00:00",
				EndTime:   "20:00:00",
			}))
			Expect(req.EffectiveScope).To(Equal(&alertstrategy.EffectiveScope{
				Type:     alertstrategy.EffectiveScopeEnvType,
				EnvTypes: []string{"production"},
				EnvIDs:   []bson.ObjectID{envID},
			}))
			Expect(req.NoticeGroupIDs).To(Equal([]int64{2001}))
			Expect(req.Enabled).To(Equal(&enabled))
			Expect(req.Operator).To(Equal("tester"))
		})

		It("returns error for invalid env id", func() {
			scope := &EffectiveScopeInput{Type: "specific_envs", EnvIDs: []string{"bad-id"}}

			_, err := UpdateAlertStrategyBody{EffectiveScope: scope}.ToUpdateReq("tester")

			Expect(err).To(HaveOccurred())
		})
	})

	Describe("AlertQueryInput", func() {
		It("converts alert id and description to search input", func() {
			input := AlertQueryInput{
				Page:        1,
				PageSize:    10,
				AlertID:     "178583503311905195",
				Description: "memory_request_usage_high",
			}

			searchInput := input.ToSearchInput()

			Expect(searchInput.AlertID).To(Equal("178583503311905195"))
			Expect(searchInput.Description).To(Equal("memory_request_usage_high"))
		})

		It("uses required paging binding tags", func() {
			typ := reflect.TypeOf(AlertQueryInput{})

			pageField, ok := typ.FieldByName("Page")
			Expect(ok).To(BeTrue())
			Expect(pageField.Tag.Get("binding")).To(Equal("required,gte=1"))

			pageSizeField, ok := typ.FieldByName("PageSize")
			Expect(ok).To(BeTrue())
			Expect(pageSizeField.Tag.Get("binding")).To(Equal("required,oneof=5 10 20 50 100"))
		})

		It("keeps explicit paging and ordering unchanged", func() {
			input := &AlertQueryInput{
				Page:      2,
				PageSize:  50,
				Ordering:  []string{"-latest_time"},
				AlertName: "cpu high",
			}

			input.Normalize()

			Expect(input.Page).To(Equal(2))
			Expect(input.PageSize).To(Equal(50))
			Expect(input.Ordering).To(Equal([]string{"-latest_time"}))
		})

		It("fills default ordering when empty", func() {
			input := &AlertQueryInput{Page: 1, PageSize: 20}

			input.Normalize()

			Expect(input.Ordering).To(Equal([]string{"-create_time"}))
		})
	})
})
