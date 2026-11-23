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

// Package resname 提供统一的环境/泳道作用域基础资源名生成与资源名合法性校验。
// 调用方通过 Build 获取作用域化后的基础名称，自行决定直接作为工作负载/CR 名称使用，或继续拼接各模块自有的附属后缀。
package resname

import (
	"crypto/rand"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/validation"

	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
)

const (
	// 共用命名空间时，附加命名以字母作前缀，确保满足 K8s 资源命名规则（字母开头）。
	// Q: 为什么不使用后缀？
	// A: 目前大部分派生资源名是在基础名称后面拼固定后缀得到的，
	//    如 <base>-plain-cfg、<base>-plain-direct、<base>-devmode-scripts，
	//    如果再加后缀，容易导致后缀拼接顺序不一致；同时外部配日志采集、监控时通常以前缀过滤。
	envPrefixLead    = "f"
	envPrefixRandLen = 4
	envPrefixCharset = "abcdefghijklmnopqrstuvwxyz0123456789"
	nameSeparator    = "-"

	// longestFixedDerivedSuffix 是从应用基础资源名固定派生的资源名中最长的后缀（开发模式脚本 ConfigMap 及同名 Volume）。
	// 北极星等按需配置的附属资源名更长，由对应扩展包在加入生效范围时调用 Validate 另行校验。
	longestFixedDerivedSuffix = "-devmode-scripts"
)

type buildOptions struct {
	trafficLaneName string
}

// Option 定义 Build 生成资源名时的可选作用域修饰项。
type Option func(*buildOptions)

// WithTrafficLane 指定流量泳道作用域；laneName 为空时不附加泳道前缀。
func WithTrafficLane(laneName string) Option {
	return func(o *buildOptions) {
		o.trafficLaneName = laneName
	}
}

// GenEnvPrefix 生成 "f" + 4 位 [a-z0-9] 的随机环境资源前缀。
func GenEnvPrefix() (string, error) {
	buf := make([]byte, envPrefixRandLen)
	if _, err := rand.Read(buf); err != nil {
		return "", errors.Wrap(err, "read random bytes")
	}
	for i, b := range buf {
		buf[i] = envPrefixCharset[int(b)%len(envPrefixCharset)]
	}
	return envPrefixLead + string(buf), nil
}

// Build 生成 baseName 在目标环境（及可选流量泳道）作用域下的集群基础资源名。
// 拼接顺序固定为：[<trafficLaneName>-][<resourcePrefix>-]<baseName>。
// 上层调用方自行决定将该结果直接作为工作负载/CR 名称，或在此基础上继续拼接模块自有的附属资源后缀。
func Build(env *envmodel.Environment, baseName string, opts ...Option) string {
	if baseName == "" {
		return ""
	}
	var options buildOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}

	name := baseName
	if env != nil && env.ResourcePrefix != "" {
		name = env.ResourcePrefix + nameSeparator + name
	}
	if options.trafficLaneName != "" {
		name = options.trafficLaneName + nameSeparator + name
	}
	return name
}

// TrimEnvPrefix 剥离集群资源名上的环境前缀，还原为原始配置名。
func TrimEnvPrefix(env *envmodel.Environment, name string) string {
	if env == nil || env.ResourcePrefix == "" {
		return name
	}
	return strings.TrimPrefix(name, env.ResourcePrefix+nameSeparator)
}

// ValidateAppNameForSharedNS 校验应用在共享命名空间环境下派生的基础资源名是否满足集群资源命名长度上限。
func ValidateAppNameForSharedNS(appName string) error {
	prefixLen := len(envPrefixLead) + envPrefixRandLen
	nameLen := prefixLen + len(nameSeparator) + len(appName) + len(longestFixedDerivedSuffix)
	if nameLen > validation.DNS1035LabelMaxLength {
		return errors.Errorf(
			"app name %s is too long to share namespace: prefixed resource names would reach %d characters (max %d)",
			appName, nameLen, validation.DNS1035LabelMaxLength,
		)
	}
	return nil
}

// Validate 校验集群资源名称的合法性（满足 K8s 通用资源命名的格式与长度要求）。
func Validate(name string) error {
	// 采用最严格的通用子集（DNS-1035 label：字母开头、小写字母/数字/连字符、最长 63 字符），
	// 同时满足 Service、Volume、Label Value 及各类工作负载/CRD 名称要求，调用方无需感知底层 RFC 细节。
	if errs := validation.IsDNS1035Label(name); len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}
