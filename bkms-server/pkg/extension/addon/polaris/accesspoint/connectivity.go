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
	"net"
	"net/http"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/tidwall/gjson"

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

	// tokenMissingWarning 未填写 Token 且北极星开启了控制台鉴权时的风险提示
	tokenMissingWarning = "token is empty, services created by the platform can be modified by anyone"
)

// errForbiddenAddress 北极星地址指向了不允许访问的地址，见 isForbiddenIP
var errForbiddenAddress = errors.New("loopback, link-local and unspecified addresses are not allowed")

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

// probeConnectivity 依次检查网络、读取控制台鉴权开关、校验 Token
func probeConnectivity(ctx context.Context, baseURL, token string) *ConnectivityReport {
	ctx, cancel := context.WithTimeout(ctx, connectivityTimeout)
	defer cancel()

	cli := newOpenAPIClient(baseURL, token)
	report := &ConnectivityReport{ConnectivityResult: ConnectivityResult{CheckedAt: time.Now()}}

	// 第一步：网络。列命名空间不需要 Token，能拿到成功响应说明地址可达、北极星侧白名单已放通平台出口 IP；
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
	rejected, err := cli.verifyToken(ctx)
	if err != nil {
		return report.fail(FailureNetworkUnreachable, "failed to verify polaris token: "+describeErr(err))
	}
	if rejected != "" {
		return report.fail(FailureTokenInvalid,
			"polaris did not accept the token, please check whether it is correct ("+rejected+")")
	}
	report.OK = true
	return report
}

// networkFailureMessage 访问北极星失败时的提示
func networkFailureMessage(baseURL string, err error) string {
	return fmt.Sprintf("cannot access polaris openapi (%s): %s. "+
		"Please check the client hosts and make sure the polaris whitelist allows access from the platform",
		baseURL, describeErr(err))
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
		httpCli: newUntrustedHTTPClient(),
	}
}

// newUntrustedHTTPClient 访问用户填写地址的 HTTP 客户端。
//
// 不跟随重定向：重定向可以把请求引到任意主机、端口与路径，Go 还会把 Token 请求头带到重定向后的主机；
// 北极星 OpenAPI 本身不会重定向，收到 3xx 按非预期响应处理。
// 建立连接前按解析后的 IP 拒绝回环、链路本地（含云元数据地址）等地址，域名解析到这些地址同样拒绝；
// 不拒绝内网私有地址，自建北极星通常就部署在内网
func newUntrustedHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// 必须直连：经过代理时，CheckDialAddress 校验的是代理的地址
	transport.Proxy = nil
	transport.DialContext = (&net.Dialer{Control: CheckDialAddress}).DialContext
	return &http.Client{
		Transport: httpcli.NewTransportWith(transport),
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// CheckDialAddress 在建立连接前检查解析后的目标 IP，address 形如 ip:port
func CheckDialAddress(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return errors.Wrapf(err, "parse dial address %q", address)
	}
	if ip := net.ParseIP(host); ip == nil || isForbiddenIP(ip) {
		return errForbiddenAddress
	}
	return nil
}

// isForbiddenIP 不允许访问的地址：回环、链路本地（169.254.0.0/16 含云元数据地址、fe80::/10）、未指定地址与组播地址
func isForbiddenIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() || ip.IsMulticast()
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

// verifyToken 只读查询一个用户来校验 Token，返回 Token 未通过的原因，为空表示通过；err 只表示请求失败。
//
// 只有查询成功才算通过；Token 被禁用、不存在、服务端错误、非北极星响应等其他情况都不通过，避免把没能校验的 Token 当作有效。
//
// TODO: 明确平台需要的 Token 权限。查询用户成功不代表有业务命名空间的写权限；而按最小权限授权的子账号 Token
// 可能没有查询用户的权限（返回 401001 NotAllowedAccess），目前会被判为无效。需要确认 TSE 上子账号的实际返回，
// 或换一个与权限无关的接口来校验 Token
func (c *openAPIClient) verifyToken(ctx context.Context) (string, error) {
	status, body, err := c.get(ctx, "/core/v1/users?limit=1")
	if err != nil {
		return "", errors.Wrap(err, "verify polaris token")
	}
	if status == http.StatusOK && gjson.GetBytes(body, "code").Int() == polarisCodeSuccess {
		return "", nil
	}
	return fmt.Sprintf("status %d, %s", status, polarisResponseInfo(body)), nil
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

// polarisResponseInfo 取北极星响应中的返回码与说明。
//
// 地址是用户填写的，可能指向内网的其他服务，不是北极星的响应时不回显响应体，避免通过提示读取内网内容
func polarisResponseInfo(body []byte) string {
	if !gjson.ValidBytes(body) {
		return "not a polaris response"
	}
	code := gjson.GetBytes(body, "code")
	if code.Type != gjson.Number {
		return "not a polaris response"
	}
	const maxInfoLen = 200
	info := []rune(gjson.GetBytes(body, "info").String())
	if len(info) > maxInfoLen {
		info = append(info[:maxInfoLen], []rune("...")...)
	}
	return fmt.Sprintf("code %d, info %s", code.Int(), string(info))
}
