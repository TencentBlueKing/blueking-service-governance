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
)

// AppConfigFileService 向后兼容的门面，内嵌 AppCfgFileDefService。
// 仅覆盖签名不同的方法（如 Create），其余方法自动提升。
//
// TODO: 所有调用方迁移到 AppCfgFileDefService 后移除此门面。
type AppConfigFileService struct {
	*AppCfgFileDefService
}

// NewAppConfigFileService 内部构造分层服务。
func NewAppConfigFileService(
	fileStore AppConfigFileStore,
	defStore AppConfigFileDefStore,
	versionStore AppConfigFileVersionStore,
) *AppConfigFileService {
	base := NewBaseAppCfgFileService(defStore, fileStore, versionStore)
	inner := NewAppCfgFileDefService(base, nil)
	return &AppConfigFileService{AppCfgFileDefService: inner}
}

// Create 创建配置文件（含 def）和初始版本，返回 AppConfigFile 以保持兼容。
// 覆盖内嵌的 AppCfgFileDefService.Create（返回 *AppConfigFileWithDef）。
func (s *AppConfigFileService) Create(
	ctx context.Context,
	params CreateCfgFileParams,
) (*AppConfigFile, error) {
	wm, err := s.AppCfgFileDefService.Create(ctx, params)
	if err != nil {
		return nil, err
	}
	return &wm.AppConfigFile, nil
}
