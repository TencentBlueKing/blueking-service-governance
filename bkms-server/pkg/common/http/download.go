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

package httpresp

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	// AttachmentContentType 附件内容类型
	AttachmentContentType = "application/octet-stream"
)

var dispositionFilenameReplacer = strings.NewReplacer(
	`\`, `\\`,
	`"`, `\"`,
	"\r", "_",
	"\n", "_",
)

// BuildAttachmentDisposition 构建附件响应头。
//
// 对纯 ASCII 文件名，同时输出 RFC 6266 / RFC 5987 推荐的两部分：
//   - filename: 传统参数，兼容只识别 ASCII 文件名的客户端
//   - filename*: 真实的 UTF-8 文件名，供现代浏览器使用
//
// 对包含中文等非 ASCII 字符的文件名，只输出 filename*。这是因为部分浏览器在
// filename 与 filename* 同时存在时会优先采用 filename，反而导致下载结果无法
// 正确展示 UTF-8 文件名。
func BuildAttachmentDisposition(filename string) string {
	safeFilename := sanitizeDispositionFilename(filename)
	escapedFilename := url.PathEscape(safeFilename)
	if !isASCII(safeFilename) {
		return fmt.Sprintf(`attachment; filename*=UTF-8''%s`, escapedFilename)
	}
	return fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, safeFilename, escapedFilename)
}

func sanitizeDispositionFilename(filename string) string {
	return dispositionFilenameReplacer.Replace(filename)
}

// isASCII 判断文件名是否完全由可直接写入 filename 参数的 ASCII 字符组成。
func isASCII(filename string) bool {
	for i := 0; i < len(filename); i++ {
		if filename[i] < 0x20 || filename[i] > 0x7e {
			return false
		}
	}
	return true
}
