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

package handler

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
)

const (
	// envFileUploadFormField 是上传环境变量文件时 HTTP 表单中文件字段的名称。
	envFileUploadFormField = "file"
	// 与 parser 的 1 MiB 文本上限保持一致，避免在进入解析前读入过大的文件。
	envFileUploadMaxBytes int64 = 1 << 20
)

func readUploadedEnvFileContent(c *gin.Context) (string, error) {
	// 没传文件返回 invalid request
	fileHeader, err := c.FormFile(envFileUploadFormField)
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return "", bkerrs.New(bkerrs.ErrCodeInvalidRequest, "env file is required")
		}
		return "", bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "read uploaded env file")
	}
	// 空文件在产品语义上等价于“没有有效导入内容”，提前拦截
	if fileHeader.Size == 0 {
		return "", bkerrs.New(bkerrs.ErrCodeInvalidRequest, "env file must not be empty")
	}
	// 先利用 multipart header 上报的 Size 做一次快速拒绝。
	if fileHeader.Size > envFileUploadMaxBytes {
		return "", bkerrs.Errorf(
			bkerrs.ErrCodeInvalidRequest,
			"env file must not exceed %d bytes",
			envFileUploadMaxBytes,
		)
	}

	file, err := fileHeader.Open()
	if err != nil {
		return "", bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "open uploaded env file")
	}
	defer file.Close()

	// 再用 LimitReader 兜底，防止 header.Size 与实际内容不一致。
	contentBytes, err := io.ReadAll(io.LimitReader(file, envFileUploadMaxBytes+1))
	if err != nil {
		return "", bkerrs.Wrap(err, bkerrs.ErrCodeInvalidRequest, "read uploaded env file")
	}
	// 如果实际读取结果仍超过上限，在进入解析逻辑前统一返回请求错误。
	if int64(len(contentBytes)) > envFileUploadMaxBytes {
		return "", bkerrs.Errorf(
			bkerrs.ErrCodeInvalidRequest,
			"env file must not exceed %d bytes",
			envFileUploadMaxBytes,
		)
	}

	return string(contentBytes), nil
}
