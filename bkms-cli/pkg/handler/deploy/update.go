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

package deploy

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"gopkg.in/yaml.v3"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/constant"
)

// updateMode 更新模式
type updateMode string

const (
	// fullUpdateMode 全量更新
	fullUpdateMode updateMode = "Full"

	// configUpdateMode 配置更新
	configUpdateMode updateMode = "Config"

	// imageUpdateMode 镜像更新
	imageUpdateMode updateMode = "Image"

	// grayscaleUpdateMode 灰度更新
	grayscaleUpdateMode updateMode = "Grayscale"
)

type strategy string

const (
	// inplaceUpdateStrategy 原地更新
	inplaceUpdateStrategy strategy = "InplaceUpdate"

	// rollingUpdateStrategy 滚动更新
	rollingUpdateStrategy strategy = "RollingUpdate"
)

// appModelUpdateSpec AppModel 更新配置
type appModelUpdateSpec struct {
	// UpdateMode 更新模式
	UpdateMode updateMode `json:"updateMode" yaml:"updateMode" validate:"required"`

	// ImageTag 镜像
	ImageTag string `json:"imageTag" yaml:"imageTag"`

	// Strategy 更新策略
	// InplaceUpdate or RollingUpdate 二选一，不能为空；一般情况使用 InplaceUpdate。
	Strategy strategy `json:"strategy" yaml:"strategy"`

	// InstanceIDs 实例名称，从文件读取的字段，非最终提交给后端的字段
	// 分隔符仅支持分号(';')，这里会兼容 字符串 和 数组输入；因此，写作 any，后续需要做类型转换
	InstanceIDs any `json:"instanceIDs" yaml:"instanceIDs"`

	// fixme 暂时不支持泳道

	// podNames 实例名称，最终提交给后端的字段
	podNames []string

	// appType 应用类型 [trpc, taf]
	appType string
}

// UpdateDeploy 更新部署，支持多环境（逗号分隔）
func UpdateDeploy(ctx context.Context, workspaceID, appID, envName, updateSpecFilePath string) error {
	// 解析多环境名称
	envNames := parseEnvNames(envName)
	if len(envNames) == 0 {
		return errors.New("env name is required")
	}

	cli := client.New()

	// 校验所有环境名称合法性
	if err := validateEnvNames(ctx, cli, workspaceID, envNames); err != nil {
		return err
	}

	// 获取应用信息（只需一次）
	app, err := cli.GetAppMinimal(ctx, workspaceID, appID)
	if err != nil {
		return err
	}

	// 检查文件是否存在
	if _, err = os.Stat(updateSpecFilePath); err != nil {
		return errors.Wrapf(err, "update spec file %s not found", updateSpecFilePath)
	}

	fileContent, err := os.ReadFile(updateSpecFilePath)
	if err != nil {
		return errors.Wrapf(err, "read update spec file %s", updateSpecFilePath)
	}

	switch app.Type {
	case constant.AppTypeHelm:
		return errors.New("helm app type is not supported yet")

	case constant.AppTypeTrpc, constant.AppTypeTaf:
		spec := new(appModelUpdateSpec)
		if err = yaml.Unmarshal(fileContent, spec); err != nil {
			return errors.Wrapf(err, "parse update spec file %s", updateSpecFilePath)
		}
		spec.appType = app.Type
		if err = spec.Validate(); err != nil {
			return err
		}

		// 对每个环境依次执行更新
		var errs []string
		for _, env := range envNames {
			if updateErr := spec.handler(ctx, appID, env); updateErr != nil {
				errs = append(errs, fmt.Sprintf("env %s: %v", env, updateErr))
				fmt.Printf("✗ Deploy update failed for env: %s (%v)\n", env, updateErr)
			} else {
				fmt.Printf("✓ Deploy updated successfully for env: %s\n", env)
			}
		}

		if len(errs) > 0 {
			return errors.Errorf("deploy update failed for some envs:\n  %s", strings.Join(errs, "\n  "))
		}
		return nil

	default:
		return errors.Errorf("unknown app type: %s", app.Type)
	}
}

// handler 处理四种更新模式
// 1. 全量更新，走发布接口重新发布一次，参数：imageTag、replicas；对应页面的配置+镜像更新功能（全量更新）
// 2. 配置更新，走发布接口重新发布一次，参数：config；对应页面的仅配置更新功能（全量更新）
// 3. 镜像更新，走实例更新接口，参数：imageTag、strategy；对应页面的仅镜像更新功能（全量更新）
// 4. 灰度更新，走实例更新接口，参数：imageTag、instanceIDs(pod名称)；对应页面的灰度更新功能（灰度更新）
// NOTE: 目前通过 switch-case 来处理不同 appType 的差异，后续如果还有新增 AppType， 考虑在下层抽象一层来屏蔽差异。
func (s *appModelUpdateSpec) handler(ctx context.Context, appID, envName string) error {
	var records []client.AppModelDeployRecord
	var err error

	switch s.appType {
	case constant.AppTypeTrpc:
		records, err = client.New().ListTrpcDeployRecords(ctx, appID, envName, "", "")
	case constant.AppTypeTaf:
		records, err = client.New().ListTafDeployRecords(ctx, appID, envName, "", "")
	default:
		return errors.Errorf("unknown app type: %s", s.appType)
	}
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return errors.Errorf("app %s has no deploy record in env %s", appID, envName)
	}
	release := records[0]

	switch s.UpdateMode {
	case fullUpdateMode:
		opts := client.AppModelDeployOptions{
			ImageTag: s.ImageTag,
			Replicas: cast.ToUint64(release.Replicas),
		}
		switch s.appType {
		case constant.AppTypeTrpc:
			return client.New().CreateAppTrpcDeploy(ctx, appID, envName, opts)
		case constant.AppTypeTaf:
			return client.New().CreateAppTafDeploy(ctx, appID, envName, opts)
		default:
			return errors.Errorf("unknown app type: %s", s.appType)
		}

	case configUpdateMode:
		opts := client.AppModelDeployOptions{
			ImageTag: release.ImageTag,
			Replicas: cast.ToUint64(release.Replicas),
		}
		switch s.appType {
		case constant.AppTypeTrpc:
			return client.New().CreateAppTrpcDeploy(ctx, appID, envName, opts)
		case constant.AppTypeTaf:
			return client.New().CreateAppTafDeploy(ctx, appID, envName, opts)
		default:
			return errors.Errorf("unknown app type: %s", s.appType)
		}

	case imageUpdateMode:
		return client.New().BatchUpdateInstance(ctx, appID, envName, s.ImageTag, string(s.Strategy))

	case grayscaleUpdateMode:
		return client.New().GrayscaleUpdateInstance(ctx, appID, envName, s.ImageTag, s.podNames,
			string(inplaceUpdateStrategy))

	default:
		return errors.Errorf("unknown update mode: %s", s.UpdateMode)
	}
}

// Validate 验证
func (s *appModelUpdateSpec) Validate() error {
	if err := validator.New().Struct(s); err != nil {
		return err
	}

	switch s.UpdateMode {
	case fullUpdateMode:
		return s.imageValidate()

	case configUpdateMode:
		return nil

	case imageUpdateMode:
		if err := s.imageValidate(); err != nil {
			return err
		}
		return s.Strategy.Validate()

	case grayscaleUpdateMode:
		if err := s.podsValidate(); err != nil {
			return err
		}
		return s.imageValidate()

	default:
		return errors.Errorf("unknown update strategy: %s", s.UpdateMode)
	}
}

// imageValidate 镜像验证
// config更新模式不需要提交镜像tag，因此只能手动验证非空
func (s *appModelUpdateSpec) imageValidate() error {
	return imageValidate(&s.ImageTag)
}

// podsValidate pod验证
// 兼容处理 写作一行使用分号的模式、以及数组的模式
func (s *appModelUpdateSpec) podsValidate() error {
	pods, err := parsePodNames(s.InstanceIDs)
	if err != nil {
		return err
	}
	s.podNames = pods
	return nil
}

// Validate 更新策略
func (s strategy) Validate() error {
	switch s {
	case inplaceUpdateStrategy, rollingUpdateStrategy:
		return nil
	default:
		return errors.Errorf("invalid update strategy: %s", s)
	}
}

// imageValidate 镜像验证（包级函数，供多个 spec 复用）
func imageValidate(imageTag *string) error {
	*imageTag = strings.TrimSpace(*imageTag)
	if *imageTag == "" {
		return errors.Errorf("imageTag is empty")
	}
	return nil
}

// parsePodNames 解析实例名称（包级函数，供多个 spec 复用）
// 兼容处理 写作一行使用分号的模式、以及数组的模式
func parsePodNames(instanceIDs any) ([]string, error) {
	if instanceIDs == nil {
		return nil, errors.Errorf("instance id is empty")
	}
	var rawPods []string
	switch v := instanceIDs.(type) {
	case string:
		rawPods = strings.Split(v, ";")
	case []any:
		rawPods = cast.ToStringSlice(v)
	default:
		return nil, errors.Errorf("invalid instance ids: %T", instanceIDs)
	}

	pods := make([]string, 0, len(rawPods))
	for _, pod := range rawPods {
		if tmp := strings.TrimSpace(pod); tmp != "" {
			pods = append(pods, tmp)
		}
	}
	if len(pods) == 0 {
		return nil, errors.Errorf("instance ids is empty")
	}
	return pods, nil
}
