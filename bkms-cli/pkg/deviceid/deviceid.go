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

// Package deviceid 计算 bkms-cli 上报给 bkms-server 的设备 ID。
package deviceid

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"regexp"
	"strings"
	"sync"

	"github.com/keygen-sh/machineid"
)

// appID 是计算设备 ID 时使用的 HMAC 消息。
const appID = "bkms-cli"

// rawIDPattern 匹配可信的操作系统机器标识：
// 32 位十六进制（Linux machine-id）、UUID（macOS IOPlatformUUID、Windows MachineGuid、BSD hostid）、
// 64 位十六进制（容器 ID）。machineid 在缺少 machine-id 时会回退到 cgroup 路径，
// 可能取到 "user-1000.slice" 之类在不同机器间重复的值，这类值一律丢弃。
var rawIDPattern = regexp.MustCompile(
	`^([0-9a-f]{32}|[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}|[0-9a-f]{64})$`,
)

var get = sync.OnceValue(func() string { return compute(machineid.ID) })

// Get 返回当前机器的设备 ID，取不到可信的机器标识时返回空字符串。
//
// 设备 ID 为 HMAC-SHA256(key=机器标识, msg="bkms-cli") 的十六进制串，与 machineid.ProtectedID
// 的结果一致，不暴露原始机器标识。每个进程只计算一次，不落盘。
func Get() string {
	return get()
}

func compute(readID func() (string, error)) string {
	id, err := readID()
	if err != nil {
		slog.Debug("read machine id failed", "error", err)
		return ""
	}
	if !rawIDPattern.MatchString(strings.ToLower(id)) {
		slog.Debug("ignore untrusted machine id")
		return ""
	}
	mac := hmac.New(sha256.New, []byte(id))
	mac.Write([]byte(appID))
	return hex.EncodeToString(mac.Sum(nil))
}
