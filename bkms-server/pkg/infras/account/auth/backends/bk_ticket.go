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

// BkTicketAuthBackend 用于上云版本的用户登录和信息获取。
type BkTicketAuthBackend struct {
	// BkLoginURL 是蓝鲸统一登录服务地址，不包含任何路径信息。
	BkLoginURL string
}

// GetLoginUrl 获取登录地址。
func (b *BkTicketAuthBackend) GetLoginUrl() string {
	return fmt.Sprintf("%s/plain/", b.BkLoginURL)
}

// GetUserCredential 获取用户票据。
func (b *BkTicketAuthBackend) GetUserCredential(request *http.Request) string {
	if userToken := request.Header.Get("X-User-Bk-Ticket"); userToken != "" {
		return userToken
	}
	cookie, err := request.Cookie("bk_ticket")
	if err != nil {
		return ""
	}
	return cookie.Value
}

// GetUserInfo 获取用户信息。
func (b *BkTicketAuthBackend) GetUserInfo(ctx context.Context, userCred string) (*UserInfo, error) {
	url := fmt.Sprintf("%s/user/get_info/", b.BkLoginURL)
	client := httpcli.NewRestyClient(ctx)

	respData := map[string]any{}
	_, err := client.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{"bk_ticket": userCred}).
		ForceContentType("application/json").
		SetResult(&respData).
		Get(url)
	if err != nil {
		return nil, errors.Wrapf(err, "get user info from %s", url)
	}

	if retCode, codeErr := cast.ToIntE(respData["ret"]); codeErr != nil {
		return nil, errors.Wrapf(codeErr, "get user info api %s return code isn't integer", url)
	} else if retCode != 0 {
		return nil, errors.Errorf("failed to get user info from %s, message: %s", url, respData["msg"])
	}
	userInfo, err := parseUserInfo(url, respData)
	if err != nil {
		return nil, errors.Wrapf(err, "parse user info from %s", url)
	}
	return userInfo, nil
}

// NewBkTicketAuthBackend 创建 BkTicketAuthBackend 实例。
func NewBkTicketAuthBackend(bkLoginURL string) *BkTicketAuthBackend {
	return &BkTicketAuthBackend{BkLoginURL: bkLoginURL}
}
