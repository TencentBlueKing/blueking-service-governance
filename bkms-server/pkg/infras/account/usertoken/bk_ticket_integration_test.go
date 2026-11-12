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

package usertoken

import (
	"context"
	"os"

	. "github.com/onsi/ginkgo/v2"

	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
)

const testAPIGatewayBaseURLKey = "FOR_TEST_APIGW_BASE_URL"

// 测试运行说明：
//
// APIGatewayTokenClient 相关测试会真实请求网关接口，并无 mock，需设置以下环境变量方能正常运行：
//
// - FOR_TEST_APIGW_BASE_URL: 蓝盾 API Gateway 域名
// - FOR_TEST_BK_APP_CODE: 蓝鲸应用 Code
// - FOR_TEST_BK_APP_SECRET: 蓝鲸应用密钥
// - FOR_TEST_RTX: 用于测试的 RTX
// - FOR_TEST_BK_TICKET: 用于测试的 BK_TICKET

// 集成测试 APIGatewayTokenClient 的多个 API，乐观链路
var _ = Describe("APIGatewayTokenClient", Serial, func() {
	It("integrated case", func() {
		baseURL := os.Getenv(testAPIGatewayBaseURLKey)
		appCode := os.Getenv("FOR_TEST_BK_APP_CODE")
		appSecret := os.Getenv("FOR_TEST_BK_APP_SECRET")
		if baseURL == "" || appCode == "" || appSecret == "" {
			Skip(testAPIGatewayBaseURLKey + ", FOR_TEST_BK_APP_CODE or FOR_TEST_BK_APP_SECRET is not set")
		}

		ctx := context.Background()
		backend := NewAPIGatewayTokenClient(baseURL, appCode, appSecret)

		// Call GetToken to get the access_token
		tokenObj, err := backend.GetToken(
			ctx,
			os.Getenv("FOR_TEST_RTX"),
			map[string]string{"bk_ticket": os.Getenv("FOR_TEST_BK_TICKET")},
			"test",
			false,
		)
		if err != nil {
			Fail(err.Error())
		}
		log.Infof(ctx, "get user token finished. token: %s, error: %v", tokenObj.AccessToken, err)
		if tokenObj.RefreshToken == "" {
			Fail("refresh token is empty")
		}

		// Call GetUserInfo to get the user information
		username, err := backend.GetUserInfo(ctx, tokenObj.AccessToken)
		log.Infof(ctx, "get user info finished. username: %s, error: %v", username, err)

		// Call RefreshToken to get a fresh access_token
		refreshedTokenObj, err := backend.RefreshToken(ctx, tokenObj.RefreshToken, "test", true)
		if err != nil {
			Fail(err.Error())
		}
		log.Infof(ctx, "refresh user token finished. token %s, error: %v", refreshedTokenObj.AccessToken, err)

		// Call GetUserInfo AGAIN, to get the user information
		username, err = backend.GetUserInfo(ctx, refreshedTokenObj.AccessToken)
		log.Infof(ctx, "get user info with refreshed token finished. username: %s, error: %v", username, err)
	})
})
