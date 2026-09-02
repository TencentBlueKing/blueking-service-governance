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

// PolarisPlatformManager 封装北极星平台服务的创建、删除和实例查询。

package polaris

import (
	"context"
	"fmt"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice"
	depsvcmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
	polarisprovider "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/provider/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	polarisInfra "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/polaris"
)

// 默认等待服务就绪的超时时间
const defaultWaitReadyTimeout = 60 * time.Second

// CreatePolarisServiceParams 创建北极星服务的参数
type CreatePolarisServiceParams struct {
	PolarisName        string
	PolarisNamespace   string
	Operator           string // 北极星负责人/操作人
	WorkspaceID        string
	AppID              string
	ScopeEnvNames      []string
	EnableWeightFactor bool
}

// CreatePolarisServiceResult 创建北极星服务的结果
type CreatePolarisServiceResult struct {
	ServiceInstanceID bson.ObjectID
	Token             string
}

// UpdateServiceParams 同步到北极星服务的字段。nil 表示该字段不改。
type UpdateServiceParams struct {
	Owners *string
	// EnableWeightFactor true 写入固定公式，false 删除两项。
	EnableWeightFactor *bool
}

// PolarisPlatformManager 北极星平台服务管理器，封装创建和删除能力
type PolarisPlatformManager struct {
	svcStore           depsvcmodel.ServiceStore
	instStore          depsvcmodel.ServiceInstanceStore
	polarisConfigStore PolarisConfigStore
}

// NewPolarisPlatformManager 创建北极星平台管理器。
func NewPolarisPlatformManager(
	svcStore depsvcmodel.ServiceStore,
	instStore depsvcmodel.ServiceInstanceStore,
	polarisConfigStore PolarisConfigStore,
) *PolarisPlatformManager {
	return &PolarisPlatformManager{
		svcStore:           svcStore,
		instStore:          instStore,
		polarisConfigStore: polarisConfigStore,
	}
}

// CreateService 由平台创建北极星服务
// 该方法会同步等待服务就绪后返回
func (m *PolarisPlatformManager) CreateService(
	ctx context.Context,
	params *CreatePolarisServiceParams,
) (*CreatePolarisServiceResult, error) {
	if params.Operator == "" {
		return nil, errors.New("operator is required when creating new polaris service")
	}

	// 按需创建 ServiceManager
	svcMgr := depservice.New(m.svcStore, m.instStore, nil, nil)

	var metadata map[string]string
	if params.EnableWeightFactor {
		metadata = weightFactorMetadata()
	}

	// 调用 depservice 创建北极星服务
	instID, err := svcMgr.CreateServiceInstance(ctx, &depservice.CreateServiceInstanceParams{
		// NOTE: 依赖服务名称存在唯一索引。这里增加随机字符串，避免首次创建失败后，影响后续创建
		Name:        fmt.Sprintf("%s-%s-%s", params.PolarisNamespace, params.PolarisName, stringx.Random(5)),
		ServiceName: "polaris",
		PlanName:    "default",
		ScopeType:   depsvcmodel.ScopeTypeWorkspace,
		WorkspaceID: params.WorkspaceID,
		Operator:    auth.MustGetUser(ctx).ID,
		Params: &polarisprovider.CreateParams{
			PolarisName:      params.PolarisName,
			PolarisNamespace: params.PolarisNamespace,
			Owners:           params.Operator,
			Metadata:         metadata,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "create polaris service instance")
	}

	// 等待服务就绪
	inst, err := m.waitForInstanceReady(ctx, svcMgr, instID, defaultWaitReadyTimeout)
	if err != nil {
		return &CreatePolarisServiceResult{ServiceInstanceID: instID},
			errors.Wrap(err, "wait for service instance ready")
	}

	// 从 Credentials 或 Config 中提取 token
	token := m.extractToken(inst)
	if token == "" {
		return &CreatePolarisServiceResult{ServiceInstanceID: instID},
			errors.New("failed to get token from service instance")
	}

	return &CreatePolarisServiceResult{
		ServiceInstanceID: instID,
		Token:             token,
	}, nil
}

// waitForInstanceReady 等待服务实例就绪
func (m *PolarisPlatformManager) waitForInstanceReady(
	ctx context.Context,
	svcMgr *depservice.ServiceManager,
	instID bson.ObjectID,
	timeout time.Duration,
) (*depsvcmodel.ServiceInstance, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	log.Debugf(ctx, "waiting for polaris service instance(id:%s) to be ready, timeout: %v", instID, timeout)

	for {
		select {
		case <-ctx.Done():
			return nil, errors.Errorf("timeout waiting for service instance ready after %v", timeout)
		case <-ticker.C:
			inst, err := svcMgr.GetServiceInstance(ctx, instID)
			if err != nil {
				return nil, errors.Wrap(err, "get service instance")
			}

			switch inst.Status {
			case depsvcmodel.AvailableStatus:
				log.Debugf(ctx, "polaris service instance(id:%s) is ready", instID)
				return inst, nil
			case depsvcmodel.CreateFailedStatus, depsvcmodel.UnavailableStatus:
				return nil, errors.Errorf("service instance creation failed: %s", inst.Message)
			case depsvcmodel.ProvisioningStatus:
				// 继续等待
				log.Debugf(ctx, "polaris service instance(id:%s) is still provisioning", instID)
			default:
				log.Warnf(ctx, "unknown service instance status: %s", inst.Status)
			}
		}
	}
}

// extractToken 从服务实例中提取 token
func (m *PolarisPlatformManager) extractToken(inst *depsvcmodel.ServiceInstance) string {
	if token, ok := inst.Credentials["token"].(string); ok && token != "" {
		return token
	}
	return ""
}

// DeleteServiceParams 删除北极星服务的参数
type DeleteServiceParams struct {
	ServiceInstanceID bson.ObjectID
	AppID             string
}

// DeleteService 删除由平台创建的北极星服务
func (m *PolarisPlatformManager) DeleteService(
	ctx context.Context,
	params *DeleteServiceParams,
) error {
	if params.ServiceInstanceID.IsZero() {
		return nil // 没有关联的服务实例，无需删除
	}

	// 按需创建 ServiceManager
	svcMgr := depservice.New(m.svcStore, m.instStore, nil, nil)

	if err := svcMgr.DeleteServiceInstance(ctx, params.ServiceInstanceID); err != nil {
		return errors.Wrap(err, "delete polaris service instance")
	}

	return nil
}

// UpdateService 将平台创建的北极星服务字段同步到北极星侧，一次 PUT。
func (m *PolarisPlatformManager) UpdateService(
	ctx context.Context,
	config *PolarisConfig,
	params *UpdateServiceParams,
) error {
	update := &polarisprovider.UpdateParams{}
	if params.Owners != nil {
		update.Owners = *params.Owners
	}
	if params.EnableWeightFactor != nil {
		if *params.EnableWeightFactor {
			update.Metadata = weightFactorMetadata()
		} else {
			update.MetadataKeysToDelete = weightFactorMetadataKeys()
		}
	}

	svcMgr := depservice.New(m.svcStore, m.instStore, nil, nil)
	if err := svcMgr.UpdateServiceInstance(ctx, config.DepSvcInstID, update); err != nil {
		return errors.Wrap(err, "update polaris service")
	}
	return nil
}

// ListPolarisServiceInstances 根据应用和环境获取所有生效北极星服务的实例。
func (m *PolarisPlatformManager) ListPolarisServiceInstances(
	ctx context.Context,
	appID, envName string,
) ([]*PolarisServiceInstances, error) {
	activeConfigs, err := m.polarisConfigStore.ListByEnv(ctx, appID, envName)
	if err != nil {
		return nil, errors.Wrap(err, "list polaris configs")
	}

	result := make([]*PolarisServiceInstances, 0, len(activeConfigs))
	for _, config := range activeConfigs {
		instances, err := polarisInfra.GetInstances(ctx, config.PolarisNamespace, config.PolarisName)
		if err != nil {
			return nil, errors.Wrapf(
				err,
				"get polaris instances for service %s/%s",
				config.PolarisNamespace,
				config.PolarisName,
			)
		}
		if len(instances) == 0 {
			continue
		}
		result = append(result, &PolarisServiceInstances{
			ServiceNamespace: config.PolarisNamespace,
			ServiceName:      config.PolarisName,
			ServicePort:      config.ServicePort,
			Instances:        instances,
		})
	}
	return result, nil
}
