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

// Package auth provide login/logout command
package auth

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/howeyc/gopass"
	"github.com/pkg/browser"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/config"
	cmdutil "github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/cmd"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/console"
)

// NewLoginCmd create login command
func NewLoginCmd() *cobra.Command {
	var useAccessToken, useBkTicket bool

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login as user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if useAccessToken && useBkTicket {
				return errors.New("cannot use both access-token and bk-ticket at the same time")
			}
			if useAccessToken {
				return loginByAccessToken()
			}
			if useBkTicket {
				return loginByBkTicket()
			}
			return loginByBrowser()
		},
		Annotations: map[string]string{
			cmdutil.SkipAuthAnnotationKey: "true",
		},
	}

	cmd.Flags().BoolVar(&useAccessToken, "access-token", false, "BlueKing AccessToken")
	cmd.Flags().BoolVar(&useBkTicket, "bk-ticket", false, "BlueKing User Ticket")
	return cmd
}

// loginByBrowser 通过浏览器登录
func loginByBrowser() error {
	tokenURL := getAccessTokenURL()
	console.Tips("Now we will open your browser...")
	console.Tips("Please copy and paste the access_token from your browser.")

	// 暂停 1s 等待用户读 tips
	time.Sleep(1 * time.Second)

	if err := browser.OpenURL(tokenURL); err != nil {
		console.Tips(
			"Don't worry, you can still manually open the browser to get the access_token: %s",
			tokenURL,
		)
		return errors.Wrap(err, "open browser")
	}
	return loginByAccessToken()
}

// loginByAccessToken 通过 AccessToken 登录
func loginByAccessToken() error {
	// read access_token implicitly
	fmt.Printf(">>> AccessToken: ")
	accessToken, err := gopass.GetPasswdMasked()
	if err != nil {
		return errors.Wrap(err, "read access token from stdin")
	}
	return login(string(accessToken))
}

// loginByBkTicket 通过 bkTicket 进行登录
func loginByBkTicket() error {
	fmt.Printf(">>> Username: ")
	usernameStr, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return errors.Wrap(err, "read username from stdin")
	}
	username := strings.TrimSpace(usernameStr)

	fmt.Printf(">>> BkTicket: ")
	bkTicket, err := gopass.GetPasswdMasked()
	if err != nil {
		return errors.Wrap(err, "read bk ticket from stdin")
	}

	accessToken, err := client.New().ExchangeBkTicketForToken(username, string(bkTicket))
	if err != nil {
		return err
	}

	return login(accessToken)
}

// login 用户登录
func login(accessToken string) error {
	fmt.Printf("User login... ")
	username, err := client.New().ValidateAccessToken(accessToken)
	if err != nil {
		return errors.Wrap(err, "login with access token")
	}

	config.G.Username = username
	config.G.AccessToken = accessToken
	if err = config.G.Dump(); err != nil {
		return errors.Wrap(err, "dump config after successful login")
	}
	color.Green("Success!")
	return nil
}

// getAccessTokenURL 返回获取 AccessToken 的页面地址
func getAccessTokenURL() string {
	return config.G.BkmsBaseURL + "/user_token/token?redirect_login=true"
}
