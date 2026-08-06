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

// Package framework 提供 e2e 基础框架功能
package framework

// EnsureLoggedIn 封装"生成配置 → 登录 → 设置 workspace"的初始化流程。
// 各测试文件的 BeforeAll 只需一行调用即可完成认证初始化。
// 如果 cfg.WorkspaceID 不为空，会自动执行 workspace set。
func EnsureLoggedIn(cli *CLI, cfg *EnvConfig) {
	GenerateConfigFile(cfg, true)
	cli.RunWithStdin(cfg.Token+"\n", "login", "--access-token")
	cli.Run("workspace", "set", cfg.WorkspaceID)
}

// RunWithoutAuth 临时切换为未认证配置执行回调函数，执行完后自动恢复登录状态。
// 消除各测试文件中重复的未认证配置 + defer 恢复逻辑。
func RunWithoutAuth(cli *CLI, cfg *EnvConfig, fn func()) {
	GenerateConfigFile(cfg, false)
	defer EnsureLoggedIn(cli, cfg)
	fn()
}
