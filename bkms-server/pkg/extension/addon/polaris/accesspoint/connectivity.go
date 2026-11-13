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

package accesspoint

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/tidwall/gjson"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/httpcli"
)

const (
	// connectivityTimeout 一次连通测试（或一次实时读取）的总时间预算
	connectivityTimeout = 10 * time.Second
	// systemNamespace 北极星内置的系统命名空间，不提供给用户选择
	systemNamespace = "Polaris"
	// namespacePageSize 分页读取命名空间的页大小，北极星单页上限为 100
	namespacePageSize = 100
	// maxResponseBytes 北极星响应体的读取上限(4MB)
	// 读取的北极星地址是用户填写的，不可信任，不能无限制的读取返回的内容
	maxResponseBytes = 4 << 20

	polarisTokenHeader = "X-Polaris-Token"
	// polarisCodeSuccess 北极星 OpenAPI 的成功返回码
	polarisCodeSuccess = 200000
	// polarisCodeTokenInvalid 北极星校验 Token 失败的返回码（token verify exception）
	polarisCodeTokenInvalid = 403001

	// tokenMissingWarning 未填写 Token 且北极星开启了控制台鉴权时的风险提示
	tokenMissingWarning = "token is empty, services created by the platform can be modified by anyone" //nolint:gosec // G101: 提示文案，不是凭证
)

// errTokenRejected 北极星拒绝了 Token
var errTokenRejected = errors.New("polaris rejected the token")

// ConnectivityReport 一次连通测试的完整结果
type ConnectivityReport struct {
	ConnectivityResult
	// Namespaces 北极星上的命名空间（已过滤系统命名空间），网络不通时为空
	Namespaces []string
}

func (r *ConnectivityReport) fail(reason FailureReason, message string) *ConnectivityReport {
	r.OK, r.Reason, r.Message = false, reason, message
	return r
}

// probeConnectivity 依次检查网络、读取控制台鉴权开关、校验 Token，任一步失败即返回，不校验写权限。
//
// 任一步请求失败（含超时）都归为网络不通，具体哪一步写在 Message 里；只有北极星明确拒绝 Token 才是 Token 无效。
// 整体耗时不超过 connectivityTimeout；Token 只放在请求头中，结果的 Message 不会包含它
func probeConnectivity(ctx context.Context, baseURL, token string) *ConnectivityReport {
	ctx, cancel := context.WithTimeout(ctx, connectivityTimeout)
	defer cancel()

	cli := newOpenAPIClient(baseURL, token)
	report := &ConnectivityReport{ConnectivityResult: ConnectivityResult{CheckedAt: time.Now()}}

	// 第一步：网络。列命名空间不需要 Token，能拿到成功响应说明地址可达、北极星白名单已放通平台出口 IP；
	// 命名空间顺带返回给前端
	namespaces, err := cli.listNamespaces(ctx)
	if err != nil {
		return report.fail(FailureNetworkUnreachable, networkFailureMessage(baseURL, err))
	}
	report.Namespaces = namespaces

	// 第二步：控制台鉴权开关（consoleOpen），管控制台与 OpenAPI（8090 端口）。平台管理服务、operator 注册实例都走这一路，
	// 使用接入点的 Token。开关打开后是否真的拦截，还取决于 auth.default.strict：非严格模式下，没有绑定鉴权策略的资源
	// 不带 Token 或带无效 Token 也能读写；严格模式下这类资源同样要求有效 Token。auth/status 读不到 strict。
	// 客户端鉴权开关（clientOpen）管 SDK 访问的 8091 端口，平台不关心，不检查
	consoleOpen, err := cli.consoleAuthOpen(ctx)
	if err != nil {
		return report.fail(FailureNetworkUnreachable, "failed to read polaris auth status: "+describeErr(err))
	}
	report.ConsoleAuth = consoleOpen

	// 第三步：Token。没有 Token 时不拦截，但开启了控制台鉴权时需要提示风险：非严格模式下平台能用，
	// 但创建的服务没有绑定鉴权策略，谁都能改；严格模式下写操作会被拒绝。strict 读不到，无法区分这两种情况
	if token == "" {
		report.OK = true
		if consoleOpen {
			report.Message = tokenMissingWarning
		}
		return report
	}
	// 只用只读接口校验 Token 是否有效，不校验写权限。查询用户的接口在非严格模式下也会校验 Token，
	// 不能改用写接口加 Token 探测：非严格模式下无效 Token 也能写成功
	if err = cli.verifyToken(ctx); err != nil {
		if errors.Is(err, errTokenRejected) {
			return report.fail(FailureTokenInvalid, "polaris rejected the token, please check whether it is correct")
		}
		return report.fail(FailureNetworkUnreachable, "failed to verify polaris token: "+describeErr(err))
	}
	report.OK = true
	return report
}

// networkFailureMessage 访问北极星失败时的提示，附上需要加入北极星白名单的平台出口 IP
func networkFailureMessage(baseURL string, err error) string {
	msg := fmt.Sprintf("cannot access polaris openapi (%s): %s. "+
		"Please check the client hosts and make sure the polaris whitelist allows the platform egress IPs",
		baseURL, describeErr(err))
	if ips := config.G.Network.EgressIPs; len(ips) > 0 {
		msg += " (" + strings.Join(ips, ", ") + ")"
	}
	return msg
}

// describeErr 把访问北极星的错误转换为提示文本，超时单独说明
func describeErr(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Sprintf("request timed out after %s", connectivityTimeout)
	}
	return err.Error()
}

// openAPIClient 访问自建北极星 OpenAPI 的只读客户端
type openAPIClient struct {
	baseURL string
	token   string
	httpCli *http.Client
}

func newOpenAPIClient(baseURL, token string) *openAPIClient {
	// 超时由调用方的 ctx 控制
	return &openAPIClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		httpCli: &http.Client{Transport: httpcli.NewTransport()},
	}
}

// listNamespaces 分页读取北极星上的全部命名空间，过滤掉系统命名空间
func (c *openAPIClient) listNamespaces(ctx context.Context) ([]string, error) {
	names := []string{}
	for offset := 0; ; offset += namespacePageSize {
		path := fmt.Sprintf("/naming/v1/namespaces?offset=%d&limit=%d", offset, namespacePageSize)
		body, err := c.getOK(ctx, path)
		if err != nil {
			return nil, errors.Wrap(err, "list polaris namespaces")
		}
		page := gjson.GetBytes(body, "namespaces").Array()
		for _, ns := range page {
			if name := ns.Get("name").String(); name != "" && name != systemNamespace {
				names = append(names, name)
			}
		}
		amount := int(gjson.GetBytes(body, "amount").Int())
		if len(page) < namespacePageSize || offset+len(page) >= amount {
			return names, nil
		}
	}
}

// consoleAuthOpen 读取北极星的控制台鉴权开关
func (c *openAPIClient) consoleAuthOpen(ctx context.Context) (bool, error) {
	body, err := c.getOK(ctx, "/core/v1/auth/status")
	if err != nil {
		return false, errors.Wrap(err, "get polaris auth status")
	}
	return gjson.GetBytes(body, "optionSwitch.options.consoleOpen").Bool(), nil
}

// verifyToken 只读查询一个用户来校验 Token：北极星返回 403001 时为 errTokenRejected。
//
// 其余失败（如子账号没有查询用户的权限）不能说明 Token 无效，不拦截
func (c *openAPIClient) verifyToken(ctx context.Context) error {
	status, body, err := c.get(ctx, "/core/v1/users?limit=1")
	if err != nil {
		return errors.Wrap(err, "verify polaris token")
	}
	if gjson.GetBytes(body, "code").Int() == polarisCodeTokenInvalid {
		return errors.Wrapf(errTokenRejected, "status %d", status)
	}
	return nil
}

// getOK 发起 GET 请求，要求 HTTP 200 且北极星返回码为成功
func (c *openAPIClient) getOK(ctx context.Context, path string) ([]byte, error) {
	status, body, err := c.get(ctx, path)
	if err != nil {
		return nil, err
	}
	code := gjson.GetBytes(body, "code")
	if status != http.StatusOK || code.Int() != polarisCodeSuccess {
		return nil, errors.Errorf("unexpected response: status %d, %s", status, polarisResponseInfo(body))
	}
	return body, nil
}

func (c *openAPIClient) get(ctx context.Context, path string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return 0, nil, errors.Wrap(err, "create request")
	}
	if c.token != "" {
		req.Header.Set(polarisTokenHeader, c.token)
	}
	resp, err := c.httpCli.Do(req)
	if err != nil {
		return 0, nil, errors.Wrap(err, "do request")
	}
	defer resp.Body.Close() // nolint

	// 北极星地址是由用户填写的，不可信任，不能无限制的读取返回的内容
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return 0, nil, errors.Wrap(err, "read response body")
	}
	return resp.StatusCode, body, nil
}

// polarisResponseInfo 取北极星响应中的返回码与说明；不是北极星的 JSON 响应时截取响应体开头
func polarisResponseInfo(body []byte) string {
	if gjson.ValidBytes(body) {
		if info := gjson.GetBytes(body, "info"); info.Exists() {
			return fmt.Sprintf("code %d, info %s", gjson.GetBytes(body, "code").Int(), info.String())
		}
	}
	const maxLen = 200
	text := []rune(strings.TrimSpace(string(body)))
	if len(text) > maxLen {
		return "body " + string(text[:maxLen]) + "..."
	}
	return "body " + string(text)
}
