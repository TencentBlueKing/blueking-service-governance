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

// Package releasestrategy 管理空间级的发布策略（分批发布的批次节奏），空间下所有应用可选用
package releasestrategy

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	// ErrStrategyNotFound 发布策略不存在
	ErrStrategyNotFound = errors.New("release strategy not found")
	// ErrStrategyNameConflict 同一空间内已存在同名发布策略
	ErrStrategyNameConflict = errors.New("release strategy name already exists")
	// ErrInvalidBatchTarget 批次目标值不合法
	ErrInvalidBatchTarget = errors.New("invalid batch target")
)

// PauseMode 批次完成后的暂停方式
type PauseMode string

const (
	// PauseModeNone 不暂停，直接进入下一批
	PauseModeNone PauseMode = "none"
	// PauseModeDuration 定时暂停，到期后自动进入下一批
	PauseModeDuration PauseMode = "duration"
	// PauseModePermanent 永久暂停，需人工继续
	PauseModePermanent PauseMode = "permanent"
)

// Batch 发布策略中的一个批次
type Batch struct {
	// Target 本批"累计更新到"的实例数：正整数（如 "1"）或百分比（如 "25%"）
	Target string `bson:"target"`
	// PauseMode 本批完成后的暂停方式
	PauseMode PauseMode `bson:"pauseMode"`
	// PauseSeconds 定时暂停时长（秒），仅 PauseMode 为 duration 时有效
	PauseSeconds int `bson:"pauseSeconds,omitempty"`
}

// ReleaseStrategy 空间级发布策略
type ReleaseStrategy struct {
	ID          bson.ObjectID `bson:"_id,omitempty"`
	WorkspaceID string        `bson:"workspaceID"`
	Name        string        `bson:"name"`
	Description string        `bson:"description"`
	Batches     []Batch       `bson:"batches"`

	Creator   string    `bson:"creator"`
	CreatedAt time.Time `bson:"createdAt"`
	Updater   string    `bson:"updater"`
	UpdatedAt time.Time `bson:"updatedAt"`
}

var (
	batchCountTargetPattern   = regexp.MustCompile(`^[1-9][0-9]*$`)
	batchPercentTargetPattern = regexp.MustCompile(`^([1-9][0-9]?|100)%$`)
)

// ParseBatchTarget 解析批次目标值：正整数（≥ 1）解析为整数，百分比（1% ~ 100%）原样保留为字符串，
// 与 Rollout CR steps[].replicas 的 int-or-string 语义一致。
func ParseBatchTarget(target string) (intstr.IntOrString, error) {
	switch {
	case batchPercentTargetPattern.MatchString(target):
		return intstr.FromString(target), nil
	case batchCountTargetPattern.MatchString(target):
		count, err := strconv.ParseInt(target, 10, 32)
		if err != nil {
			return intstr.IntOrString{}, errors.Wrapf(ErrInvalidBatchTarget, "target %q: %v", target, err)
		}
		return intstr.FromInt32(int32(count)), nil
	case strings.HasSuffix(target, "%"):
		return intstr.IntOrString{}, errors.Wrapf(
			ErrInvalidBatchTarget,
			"percentage %q out of range 1%%-100%%",
			target,
		)
	default:
		return intstr.IntOrString{}, errors.Wrapf(
			ErrInvalidBatchTarget,
			"target %q is neither a positive integer nor a percentage",
			target,
		)
	}
}
