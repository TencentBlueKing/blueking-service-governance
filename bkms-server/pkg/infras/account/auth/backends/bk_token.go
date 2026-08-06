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

package backends

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/spf13/cast"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/httpcli"
)

// BkTokenAuthBackend 用于社区开源版本的用户登录和信息获取。
type BkTokenAuthBackend struct {
	// BkLoginURL 是蓝鲸统一登录服务地址，不包含任何路径信息。
	BkLoginURL string
}

// GetLoginUrl 获取登录地址。
func (b *BkTokenAuthBackend) GetLoginUrl() string {
	return fmt.Sprintf("%s/plain/", b.BkLoginURL)
}

// GetUserCredential 获取用户票据。
func (b *BkTokenAuthBackend) GetUserCredential(request *http.Request) string {
	if userToken := request.Header.Get("X-User-Bk-Token"); userToken != "" {
		return userToken
	}
	cookie, err := request.Cookie("bk_token")
	if err != nil {
		return ""
	}
	return cookie.Value
}

// GetUserInfo 获取用户信息。
func (b *BkTokenAuthBackend) GetUserInfo(ctx context.Context, userCred string) (*UserInfo, error) {
	url := fmt.Sprintf("%s/accounts/get_user/", b.BkLoginURL)
	client := httpcli.NewRestyClient(ctx)

	respData := map[string]any{}
	_, err := client.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{"bk_token": userCred}).
		ForceContentType("application/json").
		SetResult(&respData).
		Get(url)
	if err != nil {
		return nil, errors.Wrapf(err, "get user info from %s", url)
	}

	if retCode, codeErr := cast.ToIntE(respData["code"]); codeErr != nil {
		return nil, errors.Wrapf(codeErr, "get user info api %s return code isn't integer", url)
	} else if retCode != 0 {
		return nil, errors.Errorf("failed to get user info from %s, message: %s", url, respData["message"])
	}
	userInfo, err := parseUserInfo(url, respData)
	if err != nil {
		return nil, errors.Wrapf(err, "parse user info from %s", url)
	}
	return userInfo, nil
}

// NewBkTokenAuthBackend 创建 BkTokenAuthBackend 实例。
func NewBkTokenAuthBackend(bkLoginURL string) *BkTokenAuthBackend {
	return &BkTokenAuthBackend{BkLoginURL: bkLoginURL}
}
