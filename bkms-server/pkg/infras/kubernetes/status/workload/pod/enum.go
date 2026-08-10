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

package pod

// 实例状态封闭枚举，作为实例列表接口 status 查询参数的唯一合法取值集合
//
// Parser.Parse 产出的状态码是开放集合：除下列已知值外，还会按容器状态动态拼出
// "Init: ExitCode 3"、"ExitCode: 137"、"Evicted"、"ImagePullBackOff" 等字符串。
// 接口层不能直接暴露这个开放集合，否则「非法取值返回 400」无从判定，因此在此收敛为
// 封闭枚举，并用 StatusOther 承接全部未列举状态，保证任何 Pod 都能被某个取值筛到
const (
	// StatusRunning 正常运行
	StatusRunning = "Running"
	// StatusPending 已创建但尚未全部容器就绪
	StatusPending = "Pending"
	// StatusSucceeded 所有容器正常退出且不再重启
	StatusSucceeded = "Succeeded"
	// StatusFailed 所有容器已退出且至少一个失败
	StatusFailed = "Failed"
	// StatusCompleted 容器已正常结束（终止原因为 Completed）
	StatusCompleted = "Completed"
	// StatusCrashLoopBackOff 容器反复崩溃重启
	StatusCrashLoopBackOff = "CrashLoopBackOff"
	// StatusError 容器异常终止（终止原因为 Error）
	StatusError = "Error"
	// StatusTerminating 已标记删除但尚未从集群消失
	StatusTerminating = "Terminating"
	// StatusNotReady 容器在运行但就绪探针未通过
	StatusNotReady = "NotReady"
	// StatusUnknown 无法判定状态（状态字段缺失或节点失联）
	StatusUnknown = "Unknown"
	// StatusOther 不属于上述任一已知状态的实际状态，如 Init: ExitCode 3、Evicted、ImagePullBackOff
	StatusOther = "Other"
)

// allStatuses 全部合法取值，顺序固定以保证接口文档与校验错误提示稳定
var allStatuses = []string{
	StatusRunning,
	StatusPending,
	StatusSucceeded,
	StatusFailed,
	StatusCompleted,
	StatusCrashLoopBackOff,
	StatusError,
	StatusTerminating,
	StatusNotReady,
	StatusUnknown,
	StatusOther,
}

// knownStatusSet 由 allStatuses 派生，供 IsValid / Normalize 做 O(1) 判定
var knownStatusSet = func() map[string]struct{} {
	set := make(map[string]struct{}, len(allStatuses))
	for _, s := range allStatuses {
		set[s] = struct{}{}
	}
	return set
}()

// AllStatuses 返回实例状态的全部合法取值，调用方不得修改返回的切片
func AllStatuses() []string {
	return allStatuses
}

// IsValid 判断取值是否属于实例状态封闭枚举，用于 status 查询参数校验
func IsValid(code string) bool {
	_, ok := knownStatusSet[code]
	return ok
}

// Normalize 把 Parser.Parse 产出的开放状态码归一到封闭枚举：
// 命中枚举则原样返回，其余一律归为 StatusOther，空串归为 StatusUnknown
func Normalize(code string) string {
	if code == "" {
		return StatusUnknown
	}
	if IsValid(code) {
		return code
	}
	return StatusOther
}
