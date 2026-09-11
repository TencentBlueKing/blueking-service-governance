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

package appcfg

import (
	"context"

	"github.com/pkg/errors"
)

// --- 环境文件详情 ---

// GetEnvFileDetail 获取指定环境下的文件详情，通过 ConfigKindPolicy 驱动差异行为：
//
//   - envName 为空 / 统一配置 → DisplayFile = DefaultFile
//   - policy.IsEffectiveForEnv == false → 报错（文件在该环境不生效）
//   - 有环境实例 → DisplayFile = EnvFile
//   - 无实例 + overwrite 策略 → DisplayFile = DefaultFile（默认即生效内容）
//   - 无实例 + overlay 策略 → DisplayFile = nil（无定制内容）
func (s *AppCfgFileDefService) GetEnvFileDetail(
	ctx context.Context,
	def *AppConfigFileDef,
	envName string,
) (*EnvFileDetailResult, error) {
	policy, defaultFile, envFile, err := s.loadPolicyAndEnvFiles(ctx, def, envName)
	if err != nil {
		return nil, err
	}

	result := &EnvFileDetailResult{Def: def, DefaultFile: defaultFile, DisplayFile: defaultFile}

	// 未指定环境：直接返回默认文件
	if envName == EnvNameDefault {
		s.fillEditorInfo(ctx, result)
		return result, nil
	}

	// 策略校验：该文件是否在指定环境生效（无论统一/独立配置，都必须先过挂载范围检查）
	if !policy.IsEffectiveForEnv(def, envName) {
		return nil, errors.Wrapf(ErrInvalidConfigSpec, "file %s is not effective for env %s", def.Name, envName)
	}

	// 统一配置：内容与默认一致，直接返回默认文件
	if !def.EnvConfigMode.IsIndependent() {
		s.fillEditorInfo(ctx, result)
		return result, nil
	}

	strategy := policy.GetEnvInstanceStrategy()
	if envFile != nil {
		result.HasEnvInstance = true
		result.DisplayFile = envFile
	} else if strategy == EnvInstanceStrategyOverwrite {
		// overwrite 策略（如 plain）：无实例时默认文件即为生效内容
		result.DisplayFile = defaultFile
	} else if strategy == EnvInstanceStrategyOverlay {
		// overlay 策略（如 framework）：无实例表示无定制，内容为空
		result.DisplayFile = nil
	} else {
		return nil, errors.Errorf("unsupported env instance strategy: %s", strategy)
	}

	s.fillEditorInfo(ctx, result)
	return result, nil
}

// loadPolicyAndEnvFiles 返回单个 def 在目标环境下做后续决策所需的 policy、默认文件和环境实例。
// envFile 仅在"非默认环境 + 独立配置 + 策略上对该环境生效"时才会查询；其余场景固定为 nil。
func (s *AppCfgFileDefService) loadPolicyAndEnvFiles(
	ctx context.Context,
	def *AppConfigFileDef,
	envName string,
) (
	policy ConfigKindPolicy,
	defaultFile *AppConfigFile,
	envFile *AppConfigFile,
	err error,
) {
	policy, err = s.policyFor(def.ConfigKind)
	if err != nil {
		return nil, nil, nil, err
	}

	defaultFile, err = s.FileStore.GetByDefIDAndEnv(ctx, def.ID, EnvNameDefault)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "loading default file")
	}

	// 默认环境、统一配置、或环境不在生效范围内，都不需要加载环境实例
	if envName == EnvNameDefault || !def.EnvConfigMode.IsIndependent() ||
		!policy.IsEffectiveForEnv(def, envName) {
		return policy, defaultFile, nil, nil
	}

	envFile, err = s.FindEnvInstance(ctx, def.ID, envName)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "loading env file")
	}
	return policy, defaultFile, envFile, nil
}

// fillEditorInfo 为详情结果填充编辑器相关信息（editableContentField / baseContentInfo）。
// DisplayFile 为 nil 时（overlay 无实例），标记为不可编辑。
func (s *AppCfgFileDefService) fillEditorInfo(ctx context.Context, result *EnvFileDetailResult) {
	if result.DisplayFile == nil {
		result.EditableContentField = string(EditableContentFieldNone)
		return
	}
	editor, err := NewAppConfigFileEditor(s.FileStore, s.DefStore, result.DisplayFile)
	if err != nil {
		result.EditableContentField = string(EditableContentFieldNone)
		return
	}
	result.EditableContentField = string(editor.GetEditableContentField())

	provider, err := NewBaseContentProvider(s.FileStore, s.DefStore, result.DisplayFile)
	if err != nil {
		return
	}
	info, pErr := provider.GetInfo(ctx)
	if pErr != nil {
		// ErrBaseContentEmpty 表示无 base（normal local 文件），不是异常
		return
	}
	result.BaseContentInfo = info
}

// --- 挂载预览 ---

// MountPreviewItem 挂载预览中的单条文件信息。
type MountPreviewItem struct {
	Def           AppConfigFileDef
	ContentSource AppConfigFileType // normal / overlay / overwrite
	HasEnvFile    bool              // 该环境是否有独立的文件实例
}

// GetMountPreview 获取指定环境下最终会挂载到容器内的所有配置文件及其内容来源。
func (s *AppCfgFileDefService) GetMountPreview(
	ctx context.Context,
	appID, envName string,
) ([]MountPreviewItem, error) {
	defs, err := s.DefStore.ListByApp(ctx, appID)
	if err != nil {
		return nil, errors.Wrap(err, "listing defs for mount preview")
	}

	result := make([]MountPreviewItem, 0, len(defs))
	for _, def := range defs {
		policy, pErr := s.policyFor(def.ConfigKind)
		if pErr != nil {
			continue
		}
		// 策略判断该 def 在目标环境是否生效
		if !policy.IsEffectiveForEnv(&def, envName) {
			continue
		}

		item := MountPreviewItem{Def: def, ContentSource: AppConfigFileTypeNormal, HasEnvFile: false}

		// 查找该环境是否有独立实例
		envFile, fErr := s.FileStore.GetByDefIDAndEnv(ctx, def.ID, envName)
		if fErr != nil && !errors.Is(fErr, ErrAppConfigFileNotFound) {
			return nil, errors.Wrapf(fErr, "querying env instance for def %s", def.ID.Hex())
		}
		if envFile != nil {
			item.HasEnvFile = true
			item.ContentSource = envFile.Type
			// overwrite 策略下 normal 实例语义为 overwrite
			if policy.GetEnvInstanceStrategy() == EnvInstanceStrategyOverwrite &&
				envFile.Type == AppConfigFileTypeNormal {
				item.ContentSource = "overwrite"
			}
		}

		result = append(result, item)
	}
	return result, nil
}
