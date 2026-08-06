/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

package strategy

import (
	"context"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	bkmapi "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkmonitor"
)

// SyncToRemote 将告警策略的最新状态同步到蓝鲸监控远端，并回写远端引用。
func (s *Service) SyncToRemote(
	ctx context.Context,
	ws *workspace.Workspace,
	strategyID bson.ObjectID,
	operator string,
) error {
	return s.withLockedStrategy(ctx, strategyID, "get alert strategy", func(strategy *AlertStrategy) error {
		// 1. 将策略完整同步到远端，并得到新的远端引用列表。
		remoteRefs, err := s.syncStrategyStateToRemote(ctx, ws, strategy, operator)
		if err != nil {
			return err
		}
		// 2. 把远端引用回写到本地，保持本地与远端映射一致。
		if err = s.store.Update(ctx, strategyID, bson.M{"remoteRefs": remoteRefs, "updater": operator}); err != nil {
			return errors.Wrap(err, "update remoteRefs")
		}
		return nil
	})
}

// syncStrategyStateToRemote 根据策略当前生效环境和历史远端引用，构造完整远端目标并执行同步。
func (s *Service) syncStrategyStateToRemote(
	ctx context.Context,
	ws *workspace.Workspace,
	strategy *AlertStrategy,
	operator string,
) ([]RemoteStrategyRef, error) {
	// 1. 先解析策略当前实际生效的环境范围。
	targetEnvs, err := s.resolveEffectiveEnvs(ctx, strategy)
	if err != nil {
		return nil, errors.Wrap(err, "resolve effective envs")
	}

	// 2. 为所有当前生效环境构造远端目标，构造失败的环境忽略并记录日志。
	targets := make([]remoteTargetContext, 0, len(targetEnvs)+len(strategy.RemoteRefs))
	for _, env := range targetEnvs {
		target, buildErr := s.buildRemoteTargetContext(ctx, strategy, env, "")
		if buildErr != nil {
			log.Warnf(
				ctx,
				"drop env during full sync: build target for env %s failed: %v, strategyID=%s code=%s",
				env.Name, buildErr, strategy.ID.Hex(), strategy.StrategyCode,
			)
			continue
		}
		targets = append(targets, target)
	}
	// 3. 再补齐历史泳道引用对应的目标，避免全量同步时误删仍需保留的泳道策略。
	for _, ref := range strategy.RemoteRefs {
		if ref.TrafficLaneName == "" {
			continue
		}
		env, envErr := s.envStore.Get(ctx, ref.EnvID)
		if envErr != nil {
			log.Warnf(
				ctx,
				"drop stale lane ref during full sync: get env %s failed: %v, strategyID=%s code=%s",
				ref.EnvID.Hex(), envErr, strategy.ID.Hex(), strategy.StrategyCode,
			)
			continue
		}
		if !scopeMatchesEnv(strategy.EffectiveScope, *env) {
			log.Infof(
				ctx,
				"skip out-of-scope lane ref during full sync: env=%s envID=%s lane=%s strategyID=%s code=%s",
				env.Name,
				env.ID.Hex(),
				ref.TrafficLaneName,
				strategy.ID.Hex(),
				strategy.StrategyCode,
			)
			continue
		}
		target, buildErr := s.buildRemoteTargetContext(ctx, strategy, *env, ref.TrafficLaneName)
		if buildErr != nil {
			log.Warnf(
				ctx,
				"drop stale lane ref during full sync: build target for env %s (name=%s) failed: %v, strategyID=%s code=%s",
				ref.EnvID.Hex(),
				env.Name,
				buildErr,
				strategy.ID.Hex(),
				strategy.StrategyCode,
			)
			continue
		}
		targets = append(targets, target)
	}
	// 4. 统一对比并收敛远端策略状态。
	return s.reconcileRemoteStrategy(ctx, ws, strategy, targets, operator)
}

// deleteRemoteStrategies 删除策略在蓝鲸监控远端已创建的所有告警策略实例。
func (s *Service) deleteRemoteStrategies(
	ctx context.Context,
	ws *workspace.Workspace,
	strategy *AlertStrategy,
	operator string,
) error {
	// 1. 先准备远端项目 ID 和监控客户端。
	monitorProjectID, err := ws.ResolveBkMonitorProjectID()
	if err != nil {
		return errors.Wrap(err, "resolve bkMonitorProjectID")
	}
	client, err := s.newClient(operator)
	if err != nil {
		return errors.Wrap(err, "new bkmonitor client")
	}
	// 2. 汇总去重后的远端策略 ID，没有远端资源时直接返回。
	ids := uniqueRemoteStrategyIDs(strategy.RemoteRefs)
	if len(ids) == 0 {
		return nil
	}
	// 3. 调用远端删除接口，并记录删除结果。
	if err = client.DeleteAlarmStrategy(
		ctx, &bkmapi.DeleteAlarmStrategyReq{BkBizID: monitorProjectID, IDs: ids},
	); err != nil {
		return err
	}
	log.Infof(ctx,
		"deleted remote alert strategies, strategyID=%s code=%s remoteIDs=%v",
		strategy.ID.Hex(), strategy.StrategyCode, ids,
	)
	return nil
}

// SwitchEnabled 切换策略启停状态，并尽可能保持远端告警策略状态一致。
func (s *Service) SwitchEnabled(
	ctx context.Context,
	ws *workspace.Workspace,
	strategyID bson.ObjectID,
	enabled bool,
	operator string,
) error {
	return s.withLockedStrategy(ctx, strategyID, "get alert strategy", func(strategy *AlertStrategy) error {
		if enabled && len(strategy.RemoteRefs) == 0 {
			// 1. 首次启用且尚未落远端时，先按启用态完整创建远端策略。
			next := cloneStrategy(strategy)
			next.Enabled = true
			next.Updater = operator
			remoteRefs, err := s.syncStrategyStateToRemote(ctx, ws, next, operator)
			if err != nil {
				return err
			}
			if err = s.store.Update(
				ctx,
				strategyID,
				bson.M{"enabled": true, "remoteRefs": remoteRefs, "updater": operator},
			); err != nil {
				return errors.Wrap(err, "update enabled state")
			}
			return nil
		}
		// 2. 已存在远端策略时，只同步远端启停状态即可。
		remoteIDs := uniqueRemoteStrategyIDs(strategy.RemoteRefs)
		if len(remoteIDs) > 0 {
			monitorProjectID, bizErr := ws.ResolveBkMonitorProjectID()
			if bizErr != nil {
				return errors.Wrap(bizErr, "resolve bkMonitorProjectID")
			}
			client, cErr := s.newClient(operator)
			if cErr != nil {
				return errors.Wrap(cErr, "new bkmonitor client")
			}
			if switchErr := client.SwitchAlarmStrategy(ctx, &bkmapi.SwitchAlarmStrategyReq{
				BkBizID: monitorProjectID, IDs: remoteIDs, IsEnabled: enabled,
			}); switchErr != nil {
				return errors.Wrap(switchErr, "switch remote strategies")
			}
		}
		// 3. 最后回写本地启停状态。
		if err := s.store.Update(ctx, strategyID, bson.M{"enabled": enabled, "updater": operator}); err != nil {
			return errors.Wrap(err, "update enabled state")
		}
		return nil
	})
}

// SyncStrategiesForAppInEnv 将某个应用在指定环境或泳道下命中的启用策略同步到远端。
func (s *Service) SyncStrategiesForAppInEnv(
	ctx context.Context,
	ws *workspace.Workspace,
	appID string,
	envID bson.ObjectID,
	trafficLaneName string,
	operator string,
) {
	// 1. 先获取目标环境，并查询该应用下命中该环境的启用策略。
	env, err := s.envStore.Get(ctx, envID)
	if err != nil {
		log.Errorf(ctx, "get env %s for alert sync failed: %v", envID.Hex(), err)
		return
	}
	strategies, err := s.store.ListEnabledByAppMatchingEnv(ctx, ws.ID, appID, env.Type, envID)
	if err != nil {
		log.Errorf(ctx, "list alert strategies for workspace %s app %s failed: %v", ws.ID, appID, err)
		return
	}
	log.Infof(ctx,
		"sync alert strategies, workspace=%s app=%s env=%s envID=%s lane=%s matched=%d",
		ws.ID, appID, env.Name, env.ID.Hex(), trafficLaneName, len(strategies),
	)
	// 2. 逐条校验策略是否仍匹配当前环境，并执行单策略同步。
	for i := range strategies {
		strategy := &strategies[i]
		if !strategy.Enabled || !scopeMatchesEnv(strategy.EffectiveScope, *env) {
			continue
		}
		if syncErr := s.syncStrategyToEnv(ctx, ws, strategy, *env, trafficLaneName, operator); syncErr != nil {
			log.Errorf(ctx,
				"sync alert strategy failed, strategyID=%s code=%s env=%s err=%v",
				strategy.ID.Hex(), strategy.StrategyCode, env.Name, syncErr,
			)
		}
	}
}

// CleanupStrategiesForAppInEnv 清理某个应用在指定环境或泳道下不再需要保留的远端策略引用。
func (s *Service) CleanupStrategiesForAppInEnv(
	ctx context.Context,
	ws *workspace.Workspace,
	appID string,
	envID bson.ObjectID,
	trafficLaneName string,
	operator string,
) {
	// 1. 先查出所有在目标环境或泳道上已有远端引用的策略。
	strategies, err := s.store.ListByAppAndRemoteEnv(ctx, ws.ID, appID, envID, trafficLaneName)
	if err != nil {
		log.Errorf(ctx, "list alert strategies for cleanup workspace %s app %s failed: %v", ws.ID, appID, err)
		return
	}
	log.Infof(ctx,
		"cleanup alert strategies, workspace=%s app=%s envID=%s lane=%s matched=%d",
		ws.ID, appID, envID.Hex(), trafficLaneName, len(strategies),
	)
	if len(strategies) == 0 {
		return
	}
	// 2. 逐条清理对应环境或泳道引用，单条失败不影响其他策略继续清理。
	for i := range strategies {
		if cleanupErr := s.cleanupSingleStrategy(
			ctx, ws, strategies[i].ID, envID, trafficLaneName, operator,
		); cleanupErr != nil {
			log.Errorf(ctx, "cleanup strategy %s failed: %v", strategies[i].ID.Hex(), cleanupErr)
		}
	}
}

// ListRemoteStrategies 按分页查询当前工作空间在蓝鲸监控远端创建的策略列表。
func (s *Service) ListRemoteStrategies(
	ctx context.Context,
	ws *workspace.Workspace,
	operator string,
	page, pageSize int,
) (*bkmapi.SearchAlarmStrategyResp, error) {
	monitorProjectID, err := ws.ResolveBkMonitorProjectID()
	if err != nil {
		return nil, errors.Wrap(err, "resolve bkMonitorProjectID")
	}
	client, err := s.newClient(operator)
	if err != nil {
		return nil, errors.Wrap(err, "new bkmonitor client")
	}
	return client.SearchAlarmStrategy(ctx, &bkmapi.SearchAlarmStrategyReq{
		BkBizID: monitorProjectID,
		Conditions: []map[string]any{
			{"key": "label_name", "value": []string{"bkms"}},
		},
		Page:     page,
		PageSize: pageSize,
	})
}

// cleanupSingleStrategy 清理单个策略在指定环境或泳道上的远端引用，并按剩余目标重建远端状态。
func (s *Service) cleanupSingleStrategy(
	ctx context.Context,
	ws *workspace.Workspace,
	strategyID bson.ObjectID,
	envID bson.ObjectID,
	trafficLaneName string,
	operator string,
) error {
	return s.withLockedStrategy(ctx, strategyID, "re-read strategy for cleanup", func(strategy *AlertStrategy) error {
		keepRefs := make([]RemoteStrategyRef, 0, len(strategy.RemoteRefs))
		for _, ref := range strategy.RemoteRefs {
			if sameRemoteRefTarget(ref, envID, trafficLaneName) {
				continue
			}
			keepRefs = append(keepRefs, ref)
		}
		if len(keepRefs) == len(strategy.RemoteRefs) {
			return nil
		}

		// 1. 如果已经没有任何剩余引用，则直接删除全部远端策略并清空本地引用。
		if len(keepRefs) == 0 {
			if err := s.deleteRemoteStrategies(ctx, ws, strategy, operator); err != nil {
				return errors.Wrap(err, "delete remote strategies")
			}
			if err := s.store.Update(ctx, strategy.ID, bson.M{"remoteRefs": keepRefs, "updater": operator}); err != nil {
				return errors.Wrap(err, "update remote refs after delete")
			}
			return nil
		}

		// 2. 为剩余引用重建目标列表，同时顺便剔除已经失效的脏引用。
		targets := make([]remoteTargetContext, 0, len(keepRefs))
		validKeepRefs := make([]RemoteStrategyRef, 0, len(keepRefs))
		for _, ref := range keepRefs {
			keepEnv, getErr := s.envStore.Get(ctx, ref.EnvID)
			if getErr != nil {
				log.Warnf(ctx,
					"drop stale ref during cleanup: get env %s failed: %v, strategyID=%s code=%s",
					ref.EnvID.Hex(), getErr, strategy.ID.Hex(), strategy.StrategyCode,
				)
				continue
			}
			target, buildErr := s.buildRemoteTargetContext(ctx, strategy, *keepEnv, ref.TrafficLaneName)
			if buildErr != nil {
				log.Warnf(
					ctx,
					"drop stale ref during cleanup: build target for env %s (name=%s) failed: %v, strategyID=%s code=%s",
					ref.EnvID.Hex(),
					keepEnv.Name,
					buildErr,
					strategy.ID.Hex(),
					strategy.StrategyCode,
				)
				continue
			}
			targets = append(targets, target)
			validKeepRefs = append(validKeepRefs, ref)
		}
		keepRefs = validKeepRefs

		// 3. 如果剩余引用全部失效，也需要删除远端策略并清空本地状态。
		if len(keepRefs) == 0 {
			if err := s.deleteRemoteStrategies(ctx, ws, strategy, operator); err != nil {
				return errors.Wrap(err, "delete remote strategies (all stale)")
			}
			if err := s.store.Update(ctx, strategy.ID, bson.M{"remoteRefs": keepRefs, "updater": operator}); err != nil {
				return errors.Wrap(err, "update remote refs after delete (all stale)")
			}
			return nil
		}

		// 4. 根据仍然有效的目标重新收敛远端策略，并回写最新引用。
		newRefs, reconcileErr := s.reconcileRemoteStrategy(ctx, ws, strategy, targets, operator)
		if reconcileErr != nil {
			return errors.Wrap(reconcileErr, "reconcile shared remote strategy for cleanup")
		}
		if err := s.store.Update(ctx, strategy.ID, bson.M{"remoteRefs": newRefs, "updater": operator}); err != nil {
			return errors.Wrap(err, "update remote refs after reconcile")
		}
		return nil
	})
}

// syncStrategyToEnv 将单个策略在指定环境或泳道上的目标状态同步到远端。
func (s *Service) syncStrategyToEnv(
	ctx context.Context,
	ws *workspace.Workspace,
	strategy *AlertStrategy,
	env envmodel.Environment,
	trafficLaneName, operator string,
) error {
	// 1. 先构造当前环境或泳道的目标上下文，构造失败直接终止。
	currentTarget, err := s.buildRemoteTargetContext(ctx, strategy, env, trafficLaneName)
	if err != nil {
		return err
	}

	return s.withLockedStrategy(ctx, strategy.ID, "re-read strategy for sync", func(fresh *AlertStrategy) error {
		log.Infof(
			ctx,
			"sync single alert strategy (locked), strategyID=%s code=%s "+
				"workspace=%s app=%s env=%s envID=%s lane=%s remoteRefs=%d",
			fresh.ID.Hex(),
			fresh.StrategyCode,
			ws.ID,
			fresh.AppID,
			env.Name,
			env.ID.Hex(),
			trafficLaneName,
			len(fresh.RemoteRefs),
		)

		targets := make([]remoteTargetContext, 0, len(fresh.RemoteRefs)+1)
		targets = append(targets, currentTarget)
		for _, ref := range fresh.RemoteRefs {
			if sameRemoteRefTarget(ref, env.ID, trafficLaneName) {
				continue
			}
			refEnv, getErr := s.envStore.Get(ctx, ref.EnvID)
			if getErr != nil {
				log.Warnf(ctx,
					"drop stale ref during sync: get env %s failed: %v, strategyID=%s code=%s",
					ref.EnvID.Hex(), getErr, fresh.ID.Hex(), fresh.StrategyCode,
				)
				continue
			}
			target, buildErr := s.buildRemoteTargetContext(ctx, fresh, *refEnv, ref.TrafficLaneName)
			if buildErr != nil {
				log.Warnf(ctx,
					"drop stale ref during sync: build target for env %s (name=%s) failed: %v, strategyID=%s code=%s",
					ref.EnvID.Hex(), refEnv.Name, buildErr, fresh.ID.Hex(), fresh.StrategyCode,
				)
				continue
			}
			targets = append(targets, target)
		}

		// 2. 统一收敛远端策略，再把远端引用回写本地。
		newRefs, err := s.reconcileRemoteStrategy(ctx, ws, fresh, targets, operator)
		if err != nil {
			return err
		}

		if err = s.store.Update(ctx, fresh.ID, bson.M{"remoteRefs": newRefs, "updater": operator}); err != nil {
			return err
		}
		return nil
	})
}

// reconcileRemoteStrategy 根据目标集合创建、更新或删除远端告警策略，并返回最新远端引用。
func (s *Service) reconcileRemoteStrategy(
	ctx context.Context,
	ws *workspace.Workspace,
	strategy *AlertStrategy,
	targets []remoteTargetContext,
	operator string,
) ([]RemoteStrategyRef, error) {
	// 1. 先准备远端项目 ID 和监控客户端，为后续所有远端操作复用。
	monitorProjectID, err := ws.ResolveBkMonitorProjectID()
	if err != nil {
		return nil, errors.Wrap(err, "resolve bkMonitorProjectID")
	}
	client, err := s.newClient(operator)
	if err != nil {
		return nil, errors.Wrap(err, "new bkmonitor client")
	}

	// 2. 对目标集合去重；如果目标为空，则删除历史远端策略并返回空引用。
	targets = uniqRemoteTargets(targets)
	oldIDs := uniqueRemoteStrategyIDs(strategy.RemoteRefs)
	if len(targets) == 0 {
		if len(oldIDs) == 0 {
			return []RemoteStrategyRef{}, nil
		}
		if err = client.DeleteAlarmStrategy(
			ctx,
			&bkmapi.DeleteAlarmStrategyReq{BkBizID: monitorProjectID, IDs: oldIDs},
		); err != nil {
			return nil, errors.Wrap(err, "delete empty remote strategy")
		}
		return []RemoteStrategyRef{}, nil
	}

	// 3. 有目标时优先复用已有远端策略 ID，走远端保存或更新逻辑。
	remoteStrategyID := primaryRemoteStrategyID(strategy.RemoteRefs)
	strategyName := buildRemoteStrategyName(strategy)
	saveReq := s.buildSaveAlarmStrategyReq(
		strategy,
		targets,
		monitorProjectID,
		remoteStrategyID,
		strategyName,
		operator,
	)
	resp, err := client.SaveAlarmStrategy(ctx, saveReq)
	if err != nil {
		if remoteStrategyID == 0 || !isMissingRemoteStrategyErr(err) {
			return nil, err
		}
		// 4. 如果远端历史策略已丢失，则改为新建并重新建立本地映射关系。
		log.Warnf(ctx,
			"remote strategy missing (err=%v), recreate, strategyID=%s code=%s oldRemoteID=%d",
			err, strategy.ID.Hex(), strategy.StrategyCode, remoteStrategyID,
		)
		saveReq = s.buildSaveAlarmStrategyReq(
			strategy,
			targets,
			monitorProjectID,
			0,
			strategyName,
			operator,
		)
		resp, err = client.SaveAlarmStrategy(ctx, saveReq)
		if err != nil {
			return nil, errors.Wrap(err, "recreate missing remote strategy")
		}
		log.Infof(ctx,
			"recreated remote strategy, strategyID=%s code=%s newRemoteID=%d",
			strategy.ID.Hex(), strategy.StrategyCode, resp.ID,
		)
		return buildRemoteRefsFromTargets(targets, strategyName, resp.ID), nil
	}
	// 5. 保存成功后删除已经过期的旧远端策略，避免远端残留脏数据。
	if staleIDs := staleRemoteStrategyIDs(oldIDs, resp.ID); len(staleIDs) > 0 {
		if err = client.DeleteAlarmStrategy(
			ctx,
			&bkmapi.DeleteAlarmStrategyReq{BkBizID: monitorProjectID, IDs: staleIDs},
		); err != nil {
			return nil, errors.Wrap(err, "delete stale remote strategies")
		}
	}
	log.Infof(ctx,
		"reconcile remote strategy done, strategyID=%s code=%s remoteID=%d targetCount=%d",
		strategy.ID.Hex(), strategy.StrategyCode, resp.ID, len(targets),
	)
	return buildRemoteRefsFromTargets(targets, strategyName, resp.ID), nil
}

// uniqueRemoteStrategyIDs 提取并去重远端策略 ID，过滤无效 ID。
func uniqueRemoteStrategyIDs(refs []RemoteStrategyRef) []int64 {
	ids := lo.FilterMap(refs, func(ref RemoteStrategyRef, _ int) (int64, bool) {
		return ref.RemoteStrategyID, ref.RemoteStrategyID > 0
	})
	return lo.Uniq(ids)
}

// primaryRemoteStrategyID 返回首个可复用的远端策略 ID。
func primaryRemoteStrategyID(refs []RemoteStrategyRef) int64 {
	for _, ref := range refs {
		if ref.RemoteStrategyID > 0 {
			return ref.RemoteStrategyID
		}
	}
	return 0
}

// staleRemoteStrategyIDs 计算除保留 ID 外应删除的历史远端策略 ID。
func staleRemoteStrategyIDs(ids []int64, keepID int64) []int64 {
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id == 0 || id == keepID {
			continue
		}
		result = append(result, id)
	}
	return result
}

// isMissingRemoteStrategyErr 判断远端报错是否表示策略已不存在。
func isMissingRemoteStrategyErr(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "code: 3313003") || strings.Contains(errMsg, "策略配置不存在")
}

// remoteRefKey 生成环境与泳道组合对应的远端引用唯一键。
func remoteRefKey(envID bson.ObjectID, trafficLaneName string) string {
	return envID.Hex() + "::" + trafficLaneName
}

// sameRemoteRefTarget 判断远端引用是否指向同一个环境与泳道。
func sameRemoteRefTarget(ref RemoteStrategyRef, envID bson.ObjectID, trafficLaneName string) bool {
	return ref.EnvID == envID && ref.TrafficLaneName == trafficLaneName
}

// acquireStrategyLock 获取指定策略对应的互斥锁，并返回释放函数。
func acquireStrategyLock(strategyID bson.ObjectID) (*sync.Mutex, func()) {
	key := strategyID.Hex()

	strategyMu.mu.Lock()
	entry := strategyMu.entries[key]
	if entry == nil {
		entry = &strategyLockEntry{}
		strategyMu.entries[key] = entry
	}
	entry.refs++
	strategyMu.mu.Unlock()

	release := func() {
		strategyMu.mu.Lock()
		entry.refs--
		if entry.refs == 0 {
			delete(strategyMu.entries, key)
		}
		strategyMu.mu.Unlock()
	}
	return &entry.mu, release
}

// withLockedStrategy 在策略级互斥锁保护下重新读取策略，并执行给定回调。
func (s *Service) withLockedStrategy(
	ctx context.Context,
	strategyID bson.ObjectID,
	getErrMsg string,
	fn func(strategy *AlertStrategy) error,
) error {
	mu, release := acquireStrategyLock(strategyID)
	mu.Lock()
	defer release()
	defer mu.Unlock()

	strategy, err := s.store.Get(ctx, strategyID)
	if err != nil {
		return errors.Wrap(err, getErrMsg)
	}
	return fn(strategy)
}
