package bkmonitor

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

const (
	stubDefaultUserGroupID      int64 = 1001
	stubDefaultUserGroupName          = "【APM】stub-env-prod 告警组"
	stubDefaultUserGroupBkBizID int64 = -2001
)

// stubApmApps 本地开发时返回的固定 APM 应用列表
var stubApmApps = []*ApmApp{
	{
		ID:           1001,
		Token:        "stub-apm-token-001",
		BkBizID:      100001,
		AppName:      "stub-env-prod",
		Description:  "test",
		MetricConfig: &OTLPConfig{BkDataID: 10001},
		TraceConfig:  &OTLPConfig{BkDataID: 10002},
		LogConfig:    &OTLPConfig{BkDataID: 10003},
		Creator:      "stub-user",
		CreatedAt:    "2026-01-01 00:00:00",
	},
	{
		ID:           1002,
		Token:        "stub-apm-token-002",
		BkBizID:      100001,
		AppName:      "stub-env-staging",
		Description:  "test",
		MetricConfig: &OTLPConfig{BkDataID: 20001},
		TraceConfig:  &OTLPConfig{BkDataID: 20002},
		Creator:      "stub-user",
		CreatedAt:    "2026-01-02 00:00:00",
	},
}

// stubDynamicApps 动态创建的 APM 应用列表（进程级共享），用于保证 CreateApmApp/GetOrCreate
// 创建的 APM 能被后续 ListApmApp 调用找到。
var (
	// stubDynamicApps 保存运行期间动态创建的 APM 应用
	stubDynamicApps []*ApmApp
	// stubDynamicAppsMu 用于保护并发读写
	stubDynamicAppsMu sync.Mutex

	stubUserGroups   = newDefaultStubUserGroups()
	stubUserGroupsMu sync.Mutex
)

func newDefaultStubUserGroups() map[int64]*UserGroupDetail {
	return map[int64]*UserGroupDetail{
		stubDefaultUserGroupID: newStubUserGroupDetail(
			stubDefaultUserGroupID,
			stubDefaultUserGroupBkBizID,
			stubDefaultUserGroupName,
		),
	}
}

// ResetStubStateForTest resets process-wide stub state for tests.
func ResetStubStateForTest() {
	stubDynamicAppsMu.Lock()
	stubDynamicApps = nil
	stubDynamicAppsMu.Unlock()

	stubUserGroupsMu.Lock()
	stubUserGroups = newDefaultStubUserGroups()
	stubUserGroupsMu.Unlock()
}

// StubClient 测试用的蓝鲸监控 API 客户端实现，返回模拟数据
type StubClient struct {
	operator string
}

// NewStub 创建 StubClient
func NewStub(operator string) *StubClient {
	return &StubClient{operator: operator}
}

// CreateApmApp 模拟创建 APM 应用
func (s *StubClient) CreateApmApp(
	ctx context.Context,
	bkBizID int64,
	bcsProjectCode, envName, description, operator, workspaceID string,
) (*ApmApp, error) {
	log.Infof(ctx, "Stub: CreateApmApp request: bkBizID=%d, project=%s, env=%s, operator=%s",
		bkBizID, bcsProjectCode, envName, operator)

	app := &ApmApp{
		ID:          time.Now().UnixMilli(),
		Token:       fmt.Sprintf("stub-token-%s-%d", envName, time.Now().Unix()),
		BkBizID:     bkBizID,
		AppName:     envName,
		Description: description,
		Creator:     operator,
		CreatedAt:   time.Now().Format("2006-01-02 15:04:05"),
	}

	// 将动态创建的 APM 追加到共享列表，确保 ListApmApp 能找到
	stubDynamicAppsMu.Lock()
	stubDynamicApps = append(stubDynamicApps, app)
	stubDynamicAppsMu.Unlock()

	return app, nil
}

// GetApmApp 模拟获取 APM 应用详情
func (s *StubClient) GetApmApp(ctx context.Context, bkBizID, apmAppID int64, envName string) (*ApmApp, error) {
	log.Infof(ctx, "Stub: GetApmApp request: bkBizID=%d, apmAppID=%d, envName=%s", bkBizID, apmAppID, envName)

	for _, app := range s.allApps() {
		if (apmAppID > 0 && app.ID == apmAppID) || (envName != "" && app.AppName == envName) {
			return app, nil
		}
	}

	// 默认返回第一个
	return stubApmApps[0], nil
}

// GetOrCreate 模拟获取或创建 APM 应用
func (s *StubClient) GetOrCreate(
	ctx context.Context,
	bkBizID int64,
	bcsProjectCode, envName, description, operator, workspaceID string,
) (*ApmApp, error) {
	log.Infof(ctx, "Stub: GetOrCreate request: bkBizID=%d, envName=%s", bkBizID, envName)

	for _, app := range s.allApps() {
		if app.AppName == envName {
			return app, nil
		}
	}

	return s.CreateApmApp(ctx, bkBizID, bcsProjectCode, envName, description, operator, workspaceID)
}

// ListApmApp 模拟列出 APM 应用
func (s *StubClient) ListApmApp(ctx context.Context, bkBizID int64) ([]*ApmApp, error) {
	log.Infof(ctx, "Stub: ListApmApp request: bkBizID=%d", bkBizID)
	return s.allApps(), nil
}

// allApps 返回固定列表 + 动态创建列表的合集
func (s *StubClient) allApps() []*ApmApp {
	stubDynamicAppsMu.Lock()
	defer stubDynamicAppsMu.Unlock()
	result := make([]*ApmApp, 0, len(stubApmApps)+len(stubDynamicApps))
	result = append(result, stubApmApps...)
	result = append(result, stubDynamicApps...)
	return result
}

func resolveStubBkBizID(bkBizIDs []int64) int64 {
	if len(bkBizIDs) > 0 {
		return bkBizIDs[0]
	}
	return stubDefaultUserGroupBkBizID
}

// extractStubUsers 从值班编排中提取用户组详情页所需的扁平用户列表。
// 真实接口里，同一个用户可能同时出现在 arrange.Users 与 arrange.DutyUsers 中，
// 或者出现在多个编排片段里；stub 侧需要把这些来源统一拍平并按 type+id 去重，
// 以便返回稳定的 Users 字段，避免测试因重复用户产生噪音或与线上行为不一致。
func extractStubUsers(arranges []DutyArrange) []UserGroupUser {
	if len(arranges) == 0 {
		return nil
	}

	users := make([]UserGroupUser, 0)
	seen := make(map[string]struct{})
	appendUser := func(user UserGroupUser) {
		key := user.Type + "\x00" + user.ID
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		users = append(users, user)
	}

	for _, arrange := range arranges {
		for _, user := range arrange.Users {
			appendUser(user)
		}
		for _, group := range arrange.DutyUsers {
			for _, user := range group {
				appendUser(user)
			}
		}
	}

	if len(users) == 0 {
		return nil
	}
	return users
}

func newStubUserGroupDetail(id, bkBizID int64, name string) *UserGroupDetail {
	return &UserGroupDetail{
		UserGroup: UserGroup{
			ID:      id,
			Name:    name,
			BkBizID: bkBizID,
		},
	}
}

func ensureStubUserGroupLocked(id int64) *UserGroupDetail {
	detail, ok := stubUserGroups[id]
	if ok {
		return detail
	}

	detail = newStubUserGroupDetail(id, stubDefaultUserGroupBkBizID, stubDefaultUserGroupName)
	stubUserGroups[id] = detail
	return detail
}

func cloneStubUserGroupDetail(detail *UserGroupDetail) *UserGroupDetail {
	if detail == nil {
		return nil
	}
	result := *detail
	if detail.DutyArranges != nil {
		result.DutyArranges = make([]DutyArrange, len(detail.DutyArranges))
		copy(result.DutyArranges, detail.DutyArranges)
	}
	if detail.AlertNotice != nil {
		result.AlertNotice = make([]AlertNotice, len(detail.AlertNotice))
		copy(result.AlertNotice, detail.AlertNotice)
	}
	if detail.ActionNotice != nil {
		result.ActionNotice = make([]ActionNotice, len(detail.ActionNotice))
		copy(result.ActionNotice, detail.ActionNotice)
	}
	if detail.DutyRules != nil {
		result.DutyRules = make([]int64, len(detail.DutyRules))
		copy(result.DutyRules, detail.DutyRules)
	}
	return &result
}

// ListMetadataSpaceByUID 模拟根据 space_uid 获取空间
func (s *StubClient) ListMetadataSpaceByUID(ctx context.Context, uid string) (*Space, error) {
	log.Infof(ctx, "Stub: ListMetadataSpaceByUID request: uid=%s", uid)
	return &Space{
		ID:          -100001,
		SpaceTypeID: "bkci",
		SpaceID:     "stub-project-a",
		SpaceCode:   "stub0001stub0001stub0001stub0001",
		SpaceName:   "Stub 项目 A",
		SpaceUid:    uid,
		IsBcsValid:  true,
		Status:      "normal",
		Creator:     "stub-user",
		CreatedAt:   "2026-01-01 00:00:00",
	}, nil
}

// GetMetadataSpaceDetail 模拟获取空间详情
func (s *StubClient) GetMetadataSpaceDetail(ctx context.Context, bcsProjectCode string) (*Space, error) {
	log.Infof(ctx, "Stub: GetMetadataSpaceDetail request: bcsProjectCode=%s", bcsProjectCode)
	return &Space{
		ID:          -100001,
		SpaceTypeID: "bkci",
		SpaceID:     bcsProjectCode,
		SpaceCode:   "stub0001stub0001stub0001stub0001",
		SpaceName:   "Stub 项目",
		SpaceUid:    fmt.Sprintf("bkci__%s", bcsProjectCode),
		IsBcsValid:  true,
		Status:      "normal",
		Creator:     "stub-user",
		CreatedAt:   "2026-01-01 00:00:00",
	}, nil
}

// SearchUserGroups 模拟查询告警组列表
func (s *StubClient) SearchUserGroups(ctx context.Context, req *SearchUserGroupsReq) ([]*UserGroup, error) {
	log.Infof(ctx, "Stub: SearchUserGroups request: bkBizIDs=%v", req.BkBizIDs)
	bkBizID := resolveStubBkBizID(req.BkBizIDs)
	stubUserGroupsMu.Lock()
	defer stubUserGroupsMu.Unlock()
	ensureStubUserGroupLocked(stubDefaultUserGroupID)

	ids := make([]int64, 0, len(stubUserGroups))
	for id := range stubUserGroups {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	results := make([]*UserGroup, 0, len(ids))
	for _, id := range ids {
		detail := stubUserGroups[id]
		detail.BkBizID = bkBizID
		results = append(results, &UserGroup{
			ID:      detail.ID,
			Name:    detail.Name,
			BkBizID: detail.BkBizID,
		})
	}
	return results, nil
}

// SearchUserGroupDetail 模拟查询告警组详情
func (s *StubClient) SearchUserGroupDetail(
	ctx context.Context,
	req *SearchUserGroupDetailReq,
) (*UserGroupDetail, error) {
	log.Infof(ctx, "Stub: SearchUserGroupDetail request: id=%d", req.ID)
	stubUserGroupsMu.Lock()
	defer stubUserGroupsMu.Unlock()
	return cloneStubUserGroupDetail(ensureStubUserGroupLocked(req.ID)), nil
}

// SaveUserGroup 模拟保存告警组
func (s *StubClient) SaveUserGroup(ctx context.Context, req *SaveUserGroupReq) (*UserGroupDetail, error) {
	log.Infof(ctx, "Stub: SaveUserGroup request: bkBizID=%d, name=%s", req.BkBizID, req.Name)
	// req.ID 为空表示“创建新告警组”，非空表示“更新已有告警组并沿用原 ID”。
	// 这里先解析出目标 ID；若最终仍为 0，则在下面走 stub 的默认分配逻辑。
	var groupID int64
	if req.ID != nil {
		groupID = *req.ID
	}
	detail := &UserGroupDetail{
		UserGroup: UserGroup{
			ID:            groupID,
			Name:          req.Name,
			BkBizID:       req.BkBizID,
			Channels:      req.Channels,
			Timezone:      req.Timezone,
			Users:         extractStubUsers(req.DutyArranges),
			DeleteAllowed: true,
			EditAllowed:   true,
			ConfigSource:  "UI",
		},
		AlertNotice:  req.AlertNotice,
		ActionNotice: req.ActionNotice,
		DutyArranges: req.DutyArranges,
		DutyRules:    req.DutyRules,
		DutyNotice:   req.DutyNotice,
		Path:         req.Path,
	}
	if detail.ID == 0 {
		detail.ID = stubDefaultUserGroupID
	}
	stubUserGroupsMu.Lock()
	defer stubUserGroupsMu.Unlock()
	stubUserGroups[detail.ID] = detail
	return cloneStubUserGroupDetail(detail), nil
}

// DeleteUserGroup 模拟删除告警组
func (s *StubClient) DeleteUserGroup(ctx context.Context, req *DeleteUserGroupReq) error {
	log.Infof(ctx, "Stub: DeleteUserGroup request: bkBizIDs=%v, ids=%v", req.BkBizIDs, req.IDs)
	stubUserGroupsMu.Lock()
	defer stubUserGroupsMu.Unlock()
	for _, id := range req.IDs {
		delete(stubUserGroups, id)
	}
	return nil
}

// TimeSeriesUnifyQuery 统一时序数据查询
func (s *StubClient) TimeSeriesUnifyQuery(
	ctx context.Context,
	req *TimeSeriesUnifyQueryReq,
) (*TimeSeriesUnifyQueryResp, error) {
	log.Infof(ctx, "Stub: TimeSeriesUnifyQuery request: %v", req)

	return &TimeSeriesUnifyQueryResp{
		Series: []TimeSeriesData{
			{
				MetricField: "test_metric",
			},
		},
	}, nil
}

// SearchAlarmStrategy 模拟查询告警策略列表
func (s *StubClient) SearchAlarmStrategy(
	ctx context.Context,
	req *SearchAlarmStrategyReq,
) (*SearchAlarmStrategyResp, error) {
	log.Infof(ctx, "Stub: SearchAlarmStrategy request: bk_biz_id=%d", req.BkBizID)

	return &SearchAlarmStrategyResp{
		StrategyConfigList: []AlarmStrategyItem{
			{
				ID:        10001,
				BkBizID:   req.BkBizID,
				Name:      "stub-cpu-usage-high",
				Source:    "bkms",
				Scenario:  "kubernetes",
				IsEnabled: true,
			},
		},
		Total: 1,
	}, nil
}

// SaveAlarmStrategy 模拟创建或更新告警策略
func (s *StubClient) SaveAlarmStrategy(
	ctx context.Context,
	req *SaveAlarmStrategyReq,
) (*SaveAlarmStrategyResp, error) {
	log.Infof(ctx, "Stub: SaveAlarmStrategy request: bk_biz_id=%d, name=%s", req.BkBizID, req.Name)

	strategyID := req.ID
	if strategyID == 0 {
		strategyID = time.Now().UnixMilli()
	}

	return &SaveAlarmStrategyResp{
		ID:      strategyID,
		BkBizID: req.BkBizID,
		Name:    req.Name,
	}, nil
}

// SwitchAlarmStrategy 模拟批量启停告警策略
func (s *StubClient) SwitchAlarmStrategy(ctx context.Context, req *SwitchAlarmStrategyReq) error {
	log.Infof(ctx, "Stub: SwitchAlarmStrategy request: bk_biz_id=%d, ids=%v, is_enabled=%v",
		req.BkBizID, req.IDs, req.IsEnabled)
	return nil
}

// DeleteAlarmStrategy 模拟批量删除告警策略
func (s *StubClient) DeleteAlarmStrategy(ctx context.Context, req *DeleteAlarmStrategyReq) error {
	log.Infof(ctx, "Stub: DeleteAlarmStrategy request: bk_biz_id=%d, ids=%v", req.BkBizID, req.IDs)
	return nil
}

// SearchAlert 模拟查询告警事件列表
func (s *StubClient) SearchAlert(ctx context.Context, req *SearchAlertReq) (*SearchAlertResp, error) {
	log.Infof(ctx, "Stub: SearchAlert request: bk_biz_ids=%v", req.BkBizIDs)

	return &SearchAlertResp{
		Alerts: []AlertEvent{
			{
				ID:           "stub-alert-001",
				EventID:      "stub-event-001",
				AlertName:    "CPU 使用率过高",
				Assignee:     []string{"stub-owner"},
				Status:       "ABNORMAL",
				Severity:     2,
				Description:  "CPU usage exceeds 80%",
				StrategyID:   10001,
				StrategyName: "stub-cpu-usage-high",
				TargetType:   "pod",
				Target:       "stub-pod-1",
				Dimensions:   []map[string]any{{"key": "pod", "value": "stub-pod-1"}},
				CurrentValue: "91.2%",
				DataSource:   "bk_monitor",
				Content:      "容器 CPU 使用率超过阈值",
				Detail:       map[string]any{"metric": "cpu_usage"},
				RelatedInfo:  map[string]any{"cluster": "stub-cluster"},
				BeginTime:    time.Now().Add(-1 * time.Hour).Unix(),
				EndTime:      time.Now().Unix(),
				LatestTime:   time.Now().Unix(),
				CreateTime:   time.Now().Add(-1 * time.Hour).Unix(),
			},
		},
		Total: 1,
	}, nil
}

// GetAlertDetail 模拟查询告警详情
func (s *StubClient) GetAlertDetail(ctx context.Context, req *AlertDetailReq) (map[string]any, error) {
	log.Infof(ctx, "Stub: GetAlertDetail request: bk_biz_id=%d, id=%s", req.BkBizID, req.ID)
	return map[string]any{
		"id":            req.ID,
		"event_id":      "stub-event-001",
		"alert_name":    "CPU 使用率过高",
		"strategy_name": "stub-cpu-usage-high",
		"status":        "ABNORMAL",
		"severity":      2,
		"content":       "容器 CPU 使用率超过阈值",
		"current_value": "91.2%",
		"target":        "stub-pod-1",
		"dimensions":    []map[string]any{{"key": "pod", "value": "stub-pod-1"}},
		"related_info":  map[string]any{"cluster": "stub-cluster"},
	}, nil
}
