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

// BuildAttachmentDisposition 构建附件响应头
func BuildAttachmentDisposition(filename string) string {
	safeFilename := sanitizeDispositionFilename(filename)
	escapedFilename := url.PathEscape(safeFilename)
	return fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, safeFilename, escapedFilename)
}

func sanitizeDispositionFilename(filename string) string {
	return dispositionFilenameReplacer.Replace(filename)
}
