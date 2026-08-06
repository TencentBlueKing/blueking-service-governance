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

package bkci

// 蓝盾流水线构建总状态
// ref: https://github.com/TencentBlueKing/bk-ci/blob/78d7bb691a03139aaee6fef1f758cd8fbe965305/src/backend/ci/
// core/common/common-pipeline/src/main/kotlin/com/tencent/devops/common/pipeline/enums/BuildStatus.kt#L38
const (
	// StatusSucceed 成功（最终态）
	StatusSucceed = "SUCCEED"
	// StatusFailed 失败（最终态）
	StatusFailed = "FAILED"
	// StatusCanceled 取消（最终态）
	StatusCanceled = "CANCELED"
	// StatusRunning 运行中（中间状态）
	StatusRunning = "RUNNING"
	// StatusTerminate 终止（Task 最终态）待作废
	StatusTerminate = "TERMINATE"
	// StatusReviewing 审核中（Task 中间状态）
	StatusReviewing = "REVIEWING"
	// StatusReviewAbort 审核驳回（Task 最终态）
	StatusReviewAbort = "REVIEW_ABORT"
	// StatusReviewProcessed 审核通过（Task 最终态）
	StatusReviewProcessed = "REVIEW_PROCESSED"
	// StatusHeartbeatTimeout 心跳超时（最终态）
	StatusHeartbeatTimeout = "HEARTBEAT_TIMEOUT"
	// StatusPrepareEnv 准备环境中（中间状态）
	StatusPrepareEnv = "PREPARE_ENV"
	// StatusUnexec 从未执行（最终态）
	StatusUnexec = "UNEXEC"
	// StatusSkip 跳过（最终态）
	StatusSkip = "SKIP"
	// StatusQualityCheckFail 质量红线检查失败（最终态）
	StatusQualityCheckFail = "QUALITY_CHECK_FAIL"
	// StatusQueue 排队（初始状态）
	StatusQueue = "QUEUE"
	// StatusLoopWaiting 轮循等待中 互斥组抢锁轮循（中间状态）
	StatusLoopWaiting = "LOOP_WAITING"
	// StatusCallWaiting 等待回调 用于启动构建环境插件等待构建机回调启动结果（中间状态）
	StatusCallWaiting = "CALL_WAITING"
	// StatusTryFinally 不可见的后台状态（未使用）
	StatusTryFinally = "TRY_FINALLY"
	// StatusQueueTimeout 排队超时（最终态）
	StatusQueueTimeout = "QUEUE_TIMEOUT"
	// StatusExecTimeout 执行超时（最终态）
	StatusExecTimeout = "EXEC_TIMEOUT"
	// StatusQueueCache 队列待处理，瞬态。只在启动和取消过程中存在（中间状态）
	StatusQueueCache = "QUEUE_CACHE"
	// StatusRetry 重试（中间状态，仅用于 build 运行时，不展示至前端）
	StatusRetry = "RETRY"
	// StatusPause 暂停执行，等待事件（Stage / Job / Task 中间态）
	StatusPause = "PAUSE"
	// StatusStageSuccess 当Stage人工审核取消运行时，成功（Stage / Pipeline 最终态）
	StatusStageSuccess = "STAGE_SUCCESS"
	// StatusQuotaFailed 失败 (未使用）
	StatusQuotaFailed = "QUOTA_FAILED"
	// StatusDependentWaiting 依赖等待 等待依赖的job完成才会进入准备环境（Job 中间态）
	StatusDependentWaiting = "DEPENDENT_WAITING"
	// StatusQualityCheckPass 质量红线检查通过（准入准出中间态）
	StatusQualityCheckPass = "QUALITY_CHECK_PASS"
	// StatusQualityCheckWait 质量红线等待把关（准入准出中间态） 用于启动构建环境插件等待构建机回调启动结果（中间状态）
	StatusQualityCheckWait = "QUALITY_CHECK_WAIT"
	// StatusTriggerReviewing 构建触发待审核（入队列前中间态）
	StatusTriggerReviewing = "TRIGGER_REVIEWING"
	// StatusUnknown 未知状态
	StatusUnknown = "UNKNOWN"
	// StatusPollingTimeout 轮询超时
	StatusPollingTimeout = "POLLING_TIMEOUT"
	// StatusPollingBroken 轮询中断
	StatusPollingBroken = "POLLING_BROKEN"
)

// PipelineBuildStatus 蓝盾流水线构建状态
type PipelineBuildStatus string

// IsNeverRun 是否从未执行
func (s PipelineBuildStatus) IsNeverRun() bool {
	return s == StatusUnexec || s == StatusTriggerReviewing
}

// IsFinished 是否为完成（最终）状态
func (s PipelineBuildStatus) IsFinished() bool {
	return s.IsFailure() || s.IsSuccess() || s.IsCancel() || s == StatusPollingTimeout || s == StatusPollingBroken
}

// IsFailure 是否为失败状态
func (s PipelineBuildStatus) IsFailure() bool {
	return s == StatusFailed || s.IsPassiveStop() || s.IsTimeout() || s == StatusQuotaFailed
}

// IsSuccess 是否为成功状态
func (s PipelineBuildStatus) IsSuccess() bool {
	return s == StatusSucceed || s.IsSkip() || s == StatusReviewProcessed || s == StatusQualityCheckPass
}

// IsCancel 是否为取消状态
func (s PipelineBuildStatus) IsCancel() bool {
	return s == StatusCanceled
}

// IsSkip 是否为跳过
func (s PipelineBuildStatus) IsSkip() bool {
	return s == StatusSkip
}

// IsTerminate 是否为终止
func (s PipelineBuildStatus) IsTerminate() bool {
	return s == StatusTerminate
}

// IsRunning 是否为运行中
func (s PipelineBuildStatus) IsRunning() bool {
	return s == StatusRunning ||
		s == StatusLoopWaiting ||
		s == StatusReviewing ||
		s == StatusPrepareEnv ||
		s == StatusCallWaiting ||
		s.IsPause()
}

// IsReview 是否为审核中
func (s PipelineBuildStatus) IsReview() bool {
	return s == StatusReviewing || s == StatusReviewAbort || s == StatusReviewProcessed
}

// IsReadyToRun 是否为准备运行
func (s PipelineBuildStatus) IsReadyToRun() bool {
	return s == StatusQueue || s == StatusQueueCache || s == StatusRetry || s == StatusDependentWaiting
}

// IsPassiveStop 是否为被动停止
func (s PipelineBuildStatus) IsPassiveStop() bool {
	return s.IsTerminate() || s == StatusReviewAbort || s == StatusQualityCheckFail
}

// IsPause 是否为暂停
func (s PipelineBuildStatus) IsPause() bool {
	return s == StatusPause
}

// IsTimeout 是否为超时
func (s PipelineBuildStatus) IsTimeout() bool {
	return s == StatusQueueTimeout || s == StatusExecTimeout || s == StatusHeartbeatTimeout
}
