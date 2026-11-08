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

package rules

// 本文件集中存放诊断结论中面向用户的全部文案
// 结论会同时展示在 Web 面板和 CLI 上，措辞不要依赖界面布局（如「下方」「左侧」）

// 日志区说明
const (
	// LogHintPrevious 取到重启前日志时的说明
	LogHintPrevious = "以下为进程退出前的最近一次重启日志。"
	// LogHintCurrent 取到当前容器日志时的说明
	LogHintCurrent = "以下为当前容器日志。"
)

// 相关配置表格
const (
	labelFrameworkConfigFile = "框架配置文件"
	labelFrameworkConfigPath = "配置文件路径"
	labelLogReadPath         = "日志中的读取路径"
	labelStartCommand        = "启动命令"
	// hintPathMismatch 日志路径与配置路径不一致时，标红行旁的提示
	hintPathMismatch = "与配置文件路径不一致，需改为同一路径"
)

// categoryNames 分类标识对应的展示文案
var categoryNames = map[string]string{
	CategoryConfigDelivery: "配置下发",
	CategoryStartupFailure: "启动失败",
	CategoryDeployFailure:  "部署失败",
	CategoryBuildFailure:   "构建失败",
}

var (
	actionGotoFrameworkConfig = Action{ID: ActionFrameworkConfig, Label: "前往框架配置文件"}
	actionGotoDeployConfig    = Action{ID: ActionDeployConfig, Label: "前往部署配置"}
	actionGotoBuildHistory    = Action{ID: ActionBuildHistory, Label: "查看构建记录"}
	actionRedeploy            = Action{ID: ActionRedeploy, Label: "重新部署"}
)

// frameworkConfigSpec 框架配置文件读取失败；处理步骤按日志里读取路径的形态区分，见 frameworkConfigSolutions
var frameworkConfigSpec = spec{
	id:          IDFrameworkConfigNotFound,
	name:        "框架配置文件读取失败",
	categoryID:  CategoryConfigDelivery,
	confidence:  ConfidenceHigh,
	cause:       "框架读不到配置文件，容器启动即退出",
	causeDetail: "平台会把框架配置文件按你设置的路径写入容器。日志里的读取路径必须与「应用配置 → 框架配置文件」一致，否则进程会在启动时退出。",
	actions:     []Action{actionGotoFrameworkConfig, actionRedeploy},
}

// frameworkConfigSolutions 按日志中读取路径的形态给出不同的处理步骤
var frameworkConfigSolutions = struct {
	// pathMismatch 日志是绝对路径，但与配置的路径不同
	pathMismatch []string
	// relativePath 日志是相对路径，实际位置取决于工作目录
	relativePath []string
	// unknownPath 日志里没有抽到读取路径
	unknownPath []string
}{
	pathMismatch: []string{
		"将「应用配置 → 框架配置文件」中的配置文件路径与日志中的读取路径改为一致，可以改配置，也可以改代码里的读取路径",
		"修改后重新部署",
	},
	relativePath: []string{
		"代码以相对路径读取配置，实际位置取决于启动命令的工作目录，建议改为读取「应用配置 → 框架配置文件」中的绝对路径",
		"如需保留相对路径，确认启动命令的工作目录与配置文件路径匹配后重新部署",
	},
	unknownPath: []string{
		"对比日志中的报错与「应用配置 → 框架配置文件」中的文件名和路径，两者必须完全一致",
		"代码使用相对路径读取时，需与启动命令的工作目录匹配，建议统一改为绝对路径",
	},
}

// unknownSpec 实例异常但未命中更具体的规则
var unknownSpec = spec{
	id:          IDUnknown,
	name:        "未识别的异常状态",
	categoryID:  CategoryStartupFailure,
	confidence:  ConfidenceLow,
	cause:       "实例处于异常状态，但尚未匹配到更具体的规则",
	causeDetail: "请结合该实例的日志、事件和部署配置继续排查。",
	solutions: []string{
		"查看该实例的日志与事件，确认容器退出原因",
		"检查部署配置后重新部署",
	},
	actions: []Action{actionGotoDeployConfig, actionRedeploy},
}

// deployFailedSpec 部署已失败但没有匹配到更具体的原因；causeDetail 按是否还有实例区分
var deployFailedSpec = spec{
	id:         IDDeployFailed,
	name:       "未识别的部署失败",
	categoryID: CategoryDeployFailure,
	confidence: ConfidenceLow,
	cause:      "最近一次部署没有成功，尚未匹配到更具体的原因",
	solutions: []string{
		"查看该环境最新部署记录中的失败说明",
		"确认配置后重新部署",
	},
	actions: []Action{actionGotoDeployConfig, actionRedeploy},
}

const (
	deployFailedDetailWithInstances = "实例目前都已就绪，但最近一次部署没有成功。请对照部署记录说明，确认失败发生在哪一步。"
	deployFailedDetailNoInstance    = "当前环境没有实例，最近一次部署也没有成功。请对照部署记录说明，确认失败发生在哪一步。"
)

// buildFailedSpec 一键构建部署在构建阶段失败；causeDetail 按环境是否部署过区分
var buildFailedSpec = spec{
	id:         IDBuildFailed,
	name:       "构建失败",
	categoryID: CategoryBuildFailure,
	confidence: ConfidenceHigh,
	cause:      "最近一次一键构建部署在构建阶段失败，新版本没有部署",
	solutions: []string{
		"查看构建记录中失败的步骤和日志",
		"修复代码或构建配置后重新发起构建部署",
	},
	actions: []Action{actionGotoBuildHistory},
}

const (
	buildFailedDetailKeepPrevious = "构建没有产出镜像，因此没有触发部署，环境中运行的仍是上一次部署的版本。"
	buildFailedDetailNeverDeploy  = "构建没有产出镜像，因此没有触发部署，该环境尚未部署过任何版本。"
)
