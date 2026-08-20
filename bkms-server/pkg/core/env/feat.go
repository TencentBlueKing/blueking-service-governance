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

package env

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	bkmsapp "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/app"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/gvr"
	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/observability/metrics"
)

const featureEnvNamePrefix = "feat"

// 特性环境托管 namespace 使用的 labels，与 GPA / PortPool 等资源保持同一套命名约定。
const (
	FeatureEnvNSLabelWorkspaceID = "io.tencent.bkms.workspace-id"
	FeatureEnvNSLabelEnvName     = "io.tencent.bkms.env-name"
	FeatureEnvNSLabelAppID       = "io.tencent.bkms.app-id"
	FeatureEnvNSLabelController  = "io.tencent.bkms.controller"
	FeatureEnvNSControllerValue  = "bkms-feature-env"
)

// FeatureEnvService 负责特性环境领域动作。
type FeatureEnvService struct {
	environmentStore       model.EnvironmentStore
	featureEnvCounterStore model.FeatureEnvCounterStore
	namespaceInitializer   FeatureEnvNamespaceInitializer
}

// FeatureEnvNamespaceInitializer 负责初始化特性环境使用的 Kubernetes namespace。
type FeatureEnvNamespaceInitializer interface {
	// Initialize 初始化某个集群内的特定命名空间，并写入 ownerLabels 作为托管标识。
	Initialize(ctx context.Context, clusterID, namespace string, ownerLabels map[string]string) error
}

type featureEnvNamespaceInitializer struct{}

var _ FeatureEnvNamespaceInitializer = &featureEnvNamespaceInitializer{}

// NewFeatureEnvNamespaceInitializer 创建 Kubernetes namespace 初始化器。
func NewFeatureEnvNamespaceInitializer() FeatureEnvNamespaceInitializer {
	return &featureEnvNamespaceInitializer{}
}

// Initialize 在目标集群中创建特性环境独占的命名空间。
//
// 若命名空间已存在（AlreadyExists），视为成功并打 warning 日志。
func (*featureEnvNamespaceInitializer) Initialize(
	ctx context.Context, clusterID, namespace string, ownerLabels map[string]string,
) error {
	client := k8sclient.NewWithGVR(cluster.NewConfig(clusterID), gvr.NS)
	manifest := map[string]any{
		"apiVersion": "v1",
		"kind":       k8skind.NS,
		"metadata": map[string]any{
			"name":   namespace,
			"labels": maps.Clone(ownerLabels),
		},
	}

	_, err := client.Create(ctx, "", manifest, metav1.CreateOptions{})
	if errors.Is(err, k8sclient.ErrResourceAlreadyExists) {
		log.WarnAttrs(ctx, "feature env namespace already exists, treat as success",
			slog.String("cluster_id", clusterID),
			slog.String("namespace", namespace),
		)
		return nil
	}
	return err
}

// featureEnvOwnerLabels 构造特性环境 namespace 的托管 labels。
func featureEnvOwnerLabels(env *model.Environment) map[string]string {
	return map[string]string{
		FeatureEnvNSLabelWorkspaceID: env.WorkspaceID,
		FeatureEnvNSLabelEnvName:     env.Name,
		FeatureEnvNSLabelAppID:       env.OwnerAppID,
		FeatureEnvNSLabelController:  FeatureEnvNSControllerValue,
	}
}

// CreateFeatureEnvInput 描述创建特性环境所需的核心输入。
type CreateFeatureEnvInput struct {
	// App 是特性环境所属的应用
	App *bkmsapp.Application `validate:"required"`
	// SourceEnv 是特性环境的来源环境
	SourceEnv *model.Environment `validate:"required"`

	// DisplayName 是特性环境的展示名，其他的 name 等将由程序自动生成，不允许自定义
	DisplayName string `validate:"trim_not_blank"`
	// Creator 是特性环境的创建者
	Creator string
}

// NewFeatureEnvService 创建特性环境服务。
func NewFeatureEnvService(
	environmentStore model.EnvironmentStore,
	featureEnvCounterStore model.FeatureEnvCounterStore,
	namespaceInitializer FeatureEnvNamespaceInitializer,
) *FeatureEnvService {
	return &FeatureEnvService{
		environmentStore:       environmentStore,
		featureEnvCounterStore: featureEnvCounterStore,
		namespaceInitializer:   namespaceInitializer,
	}
}

// ListAppFeatEnvs 获取应用的全部特性环境及其可用的来源标准环境。
//
// Returns:
//
//	特性环境切片，map[envName]sourceEnv；由于来源环境可以独立删除，map 不保证包含每个特性环境。
func ListAppFeatEnvs(
	ctx context.Context,
	environmentStore model.EnvironmentStore,
	app *bkmsapp.Application,
) ([]model.Environment, map[string]*model.Environment, error) {
	featureEnvs, err := environmentStore.ListAppFeatEnvs(ctx, app.ID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "list app feature environments")
	}
	if len(featureEnvs) == 0 {
		return featureEnvs, map[string]*model.Environment{}, nil
	}

	// 构建来源环境的 map
	standardEnvs, err := environmentStore.ListStdEnvs(ctx, app.WorkspaceID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "list feature environment sources")
	}

	standardEnvByID := make(map[string]*model.Environment, len(standardEnvs))
	for i := range standardEnvs {
		standardEnvByID[standardEnvs[i].ID.Hex()] = &standardEnvs[i]
	}
	sourceEnvByFeatEnvName := make(map[string]*model.Environment, len(featureEnvs))
	for _, featureEnv := range featureEnvs {
		if sourceEnv, ok := standardEnvByID[featureEnv.SourceEnvID.Hex()]; ok {
			sourceEnvByFeatEnvName[featureEnv.Name] = sourceEnv
		}
	}
	return featureEnvs, sourceEnvByFeatEnvName, nil
}

// Create 根据应用、来源环境和展示名称创建特性环境记录。
//
// 关键约束：
// - sourceEnv 必须是标准环境，且与应用属于同一个 workspace；
// - name / namespace 由平台按 feat-<appID>-<index> 自动生成，调用方不能指定；
// - 集群的 namespace 也是自动生成，将由后台自动完成初始化；
func (s *FeatureEnvService) Create(ctx context.Context, input CreateFeatureEnvInput) (*model.Environment, error) {
	if err := validate.Struct(input); err != nil {
		return nil, errors.Wrap(formatError(err), "validate create feature environment input")
	}

	index, err := s.featureEnvCounterStore.Next(ctx, input.App.ID)
	if err != nil {
		return nil, errors.Wrap(err, "allocate feature environment index")
	}

	// 特性环境的内部 name 与 namespace 使用同一套规则生成，确保在工作空间内稳定唯一。
	name := fmt.Sprintf("%s-%s-%d", featureEnvNamePrefix, input.App.ID, index)
	env := &model.Environment{
		Name:        name,
		DisplayName: strings.TrimSpace(input.DisplayName),
		Type:        input.SourceEnv.Type,
		WorkspaceID: input.App.WorkspaceID,
		Kind:        model.EnvironmentKindFeature,
		OwnerAppID:  input.App.ID,
		SourceEnvID: input.SourceEnv.ID,
		Cluster: model.BizCluster{
			ProjectCode:  input.SourceEnv.Cluster.ProjectCode,
			ClusterID:    input.SourceEnv.Cluster.ClusterID,
			ClusterType:  input.SourceEnv.Cluster.ClusterType,
			Namespace:    name,
			IsFederation: input.SourceEnv.Cluster.IsFederation,
		},
		Creator: input.Creator,
	}

	envID, err := s.environmentStore.Create(ctx, env)
	if err != nil {
		return nil, errors.Wrap(err, "create feature environment")
	}
	env.ID = envID

	// 先落库再创建 namespace：避免 K8s 成功而 Mongo 失败时留下无法通过环境记录回收的孤儿 namespace。
	// namespace 初始化失败时仍返回已创建的环境记录，仅记录错误日志。
	if err = s.namespaceInitializer.Initialize(
		ctx,
		env.Cluster.ClusterID,
		env.Cluster.Namespace,
		featureEnvOwnerLabels(env),
	); err != nil {
		metrics.FeatureEnvNamespaceInitFailed()
		// TODO: 后续应更好处理此种状况（例如标记 createFailed、支持重试补建 namespace）。
		log.ErrorAttrs(ctx, "TODO: initialize feature env namespace failed",
			slog.String("env_id", env.ID.Hex()),
			slog.String("cluster_id", env.Cluster.ClusterID),
			slog.String("namespace", env.Cluster.Namespace),
			slog.Any("error", err),
		)
	}

	return env, nil
}

func validateCreateFeatureEnvInputStruct(sl validator.StructLevel) {
	input := sl.Current().Interface().(CreateFeatureEnvInput)
	if input.App == nil || input.SourceEnv == nil {
		return
	}

	if input.SourceEnv.IsFeatureEnv() {
		sl.ReportError(input.SourceEnv.Kind, "SourceEnv", "sourceEnv", "standard_env", "")
	}
	if input.SourceEnv.WorkspaceID != input.App.WorkspaceID {
		sl.ReportError(
			input.SourceEnv.WorkspaceID,
			"SourceEnv",
			"sourceEnv",
			"same_workspace",
			"",
		)
	}
}
