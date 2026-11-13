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

package accesspoint_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/polaris/accesspoint"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/misc/audit"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

const (
	validToken = "valid-token" //nolint:gosec // G101: 测试用的假 Token
	badToken   = "bad-token"   //nolint:gosec // G101: 测试用的假 Token
)

// fakePolaris 模拟北极星 OpenAPI 的三个只读接口，响应格式与腾讯云 TSE 北极星一致
type fakePolaris struct {
	mu          sync.Mutex
	namespaces  []string
	consoleOpen bool
	clientOpen  bool
	// tokens 收到的 X-Polaris-Token 请求头
	tokens []string
}

func (f *fakePolaris) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	write := func(status int, body map[string]any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
	}
	switch r.URL.Path {
	case "/naming/v1/namespaces":
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		end := min(offset+limit, len(f.namespaces))
		page := []map[string]string{}
		for _, ns := range f.namespaces[min(offset, end):end] {
			page = append(page, map[string]string{"name": ns})
		}
		write(http.StatusOK, map[string]any{
			"code": 200000, "info": "execute success", "amount": len(f.namespaces), "size": len(page),
			"namespaces": page,
		})
	case "/core/v1/auth/status":
		write(http.StatusOK, map[string]any{
			"code": 200000, "info": "execute success",
			"optionSwitch": map[string]any{"options": map[string]string{
				"auth":        strconv.FormatBool(f.consoleOpen || f.clientOpen),
				"clientOpen":  strconv.FormatBool(f.clientOpen),
				"consoleOpen": strconv.FormatBool(f.consoleOpen),
			}},
		})
	case "/core/v1/users":
		token := r.Header.Get("X-Polaris-Token")
		f.tokens = append(f.tokens, token)
		if token != validToken {
			write(http.StatusForbidden, map[string]any{"code": 403001, "info": "token verify exception"})
			return
		}
		write(http.StatusOK, map[string]any{"code": 200000, "info": "execute success", "users": []any{}})
	default:
		http.NotFound(w, r)
	}
}

func (f *fakePolaris) receivedTokens() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string{}, f.tokens...)
}

var _ = Describe("Manager", func() {
	var (
		ctx         context.Context
		manager     *accesspoint.Manager
		apStore     *accesspoint.AccessPointStore
		bindStore   *accesspoint.BindingStore
		workspaceID string

		fake *fakePolaris
		// polarisURL 被测代码访问的北极星 OpenAPI 地址，默认指向 fake
		polarisURL string
		// deadURL 一个已关闭、不可达的地址
		deadURL string

		auditMu     sync.Mutex
		auditRecord []*audit.OperationRecord
	)

	recordedAudits := func() []*audit.OperationRecord {
		auditMu.Lock()
		defer auditMu.Unlock()
		return append([]*audit.OperationRecord{}, auditRecord...)
	}

	createParams := func(name string) accesspoint.CreateParams {
		return accesspoint.CreateParams{
			Name:        name,
			ConsoleURL:  "console.example.com:8080",
			ClientHosts: []string{"10.0.0.1", "polaris.example.com"},
			Token:       validToken,
		}
	}

	mustCreate := func(name string) *accesspoint.PolarisAccessPoint {
		p, err := manager.Create(ctx, workspaceID, createParams(name))
		Expect(err).NotTo(HaveOccurred())
		return p
	}

	mustGet := func(id bson.ObjectID) *accesspoint.PolarisAccessPoint {
		p, err := apStore.Get(ctx, store.ByID(id))
		Expect(err).NotTo(HaveOccurred())
		return p
	}

	BeforeEach(func() {
		ctx = auth.WithUser(testutil.NewDefaultTenantContext(), auth.User{ID: "ws-admin"})
		app := fxtest.New(GinkgoT(), accesspoint.FxModule, fx.Populate(&manager))
		app.RequireStart()
		DeferCleanup(app.RequireStop)

		apStore = testutil.NewStore(GinkgoT(), accesspoint.AccessPointMeta)
		bindStore = testutil.NewStore(GinkgoT(), accesspoint.BindingMeta)
		workspaceID = "ws-" + stringx.Random(8)
		setPolarisConfig(config.PolarisConfig{AllowWorkspaceAccessPoints: true})

		fake = &fakePolaris{namespaces: []string{"Polaris", "Test", "Production"}}
		srv := httptest.NewServer(fake)
		DeferCleanup(srv.Close)
		dead := httptest.NewServer(http.NotFoundHandler())
		dead.Close()
		polarisURL, deadURL = srv.URL, dead.URL

		mockey.Mock(accesspoint.OpenAPIBaseURL).To(func(string) string { return polarisURL }).Build()
		auditRecord = nil
		mockey.Mock(audit.AddOperationRecordAsync).To(
			func(_ context.Context, op audit.OperationType, res audit.ResourceType, id string, opts ...audit.Option) {
				r := &audit.OperationRecord{OperationType: op, ResourceType: res, ResourceID: id}
				for _, opt := range opts {
					opt(r)
				}
				auditMu.Lock()
				defer auditMu.Unlock()
				auditRecord = append(auditRecord, r)
			}).Build()
	})

	AfterEach(func() {
		mockey.UnPatchAll()
		Expect(testutil.CleanupMetaColl(accesspoint.AccessPointMeta)).To(Succeed())
		Expect(testutil.CleanupMetaColl(accesspoint.BindingMeta)).To(Succeed())
	})

	Describe("TestConnectivity", func() {
		test := func(token *string) *accesspoint.ConnectivityReport {
			report, err := manager.TestConnectivity(ctx, workspaceID, accesspoint.ConnectivityParams{
				ClientHosts: []string{"10.0.0.1"}, Token: token,
			})
			Expect(err).NotTo(HaveOccurred())
			return report
		}

		It("should pass and return namespaces without the system namespace", func() {
			fake.consoleOpen = true

			report := test(lo.ToPtr(validToken))

			Expect(report.OK).To(BeTrue())
			Expect(report.Reason).To(BeEmpty())
			Expect(report.Message).To(BeEmpty())
			Expect(report.ConsoleAuth).To(BeTrue())
			Expect(report.Namespaces).To(Equal([]string{"Test", "Production"}))
			Expect(fake.receivedTokens()).To(Equal([]string{validToken}))
		})

		It("should read namespaces across pages", func() {
			fake.namespaces = []string{"Polaris"}
			for i := range 150 {
				fake.namespaces = append(fake.namespaces, fmt.Sprintf("ns-%03d", i))
			}

			report := test(nil)

			Expect(report.OK).To(BeTrue())
			Expect(report.Namespaces).To(HaveLen(150))
			Expect(report.Namespaces[149]).To(Equal("ns-149"))
		})

		It("should fail as network unreachable with egress IPs when polaris is unreachable", func() {
			polarisURL = deadURL
			origin := config.G.Network.EgressIPs
			config.G.Network.EgressIPs = []string{"1.1.1.1", "2.2.2.2"}
			DeferCleanup(func() { config.G.Network.EgressIPs = origin })

			report := test(lo.ToPtr(validToken))

			Expect(report.OK).To(BeFalse())
			Expect(report.Reason).To(Equal(accesspoint.FailureNetworkUnreachable))
			Expect(report.Message).To(ContainSubstring("1.1.1.1, 2.2.2.2"))
			Expect(report.Message).NotTo(ContainSubstring(validToken))
			Expect(report.Namespaces).To(BeEmpty())
		})

		It("should ignore client auth", func() {
			fake.clientOpen = true

			report := test(lo.ToPtr(validToken))

			Expect(report.OK).To(BeTrue())
			Expect(report.Reason).To(BeEmpty())
		})

		It("should fail as token invalid when polaris rejects the token", func() {
			report := test(lo.ToPtr(badToken))

			Expect(report.OK).To(BeFalse())
			Expect(report.Reason).To(Equal(accesspoint.FailureTokenInvalid))
			Expect(report.Message).NotTo(ContainSubstring(badToken))
		})

		It("should pass with a risk warning when token is empty and console auth is open", func() {
			fake.consoleOpen = true

			report := test(lo.ToPtr(""))

			Expect(report.OK).To(BeTrue())
			Expect(report.Message).To(ContainSubstring("token is empty"))
			Expect(fake.receivedTokens()).To(BeEmpty())
		})

		It("should pass without warning when token is omitted and console auth is closed", func() {
			report := test(nil)

			Expect(report.OK).To(BeTrue())
			Expect(report.Message).To(BeEmpty())
		})

		It("should use the saved token and record the result for an existing access point", func() {
			p := mustCreate("tse")
			createdAt := mustGet(p.ID).UpdatedAt
			fakeURL := polarisURL
			polarisURL = deadURL

			report, err := manager.TestConnectivity(ctx, workspaceID, accesspoint.ConnectivityParams{
				ID: p.ID.Hex(), ClientHosts: []string{"10.0.0.2"},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(report.Reason).To(Equal(accesspoint.FailureNetworkUnreachable))
			got := mustGet(p.ID)
			Expect(got.LastConnectivity.OK).To(BeFalse())
			Expect(got.LastConnectivity.Reason).To(Equal(accesspoint.FailureNetworkUnreachable))
			// 记录连通结果不算修改接入点
			Expect(got.UpdatedAt).To(BeTemporally("==", createdAt))
			Expect(got.Token).To(Equal(validToken))

			polarisURL = fakeURL
			_, err = manager.TestConnectivity(ctx, workspaceID, accesspoint.ConnectivityParams{
				ID: p.ID.Hex(), ClientHosts: []string{"10.0.0.2"},
			})
			Expect(err).NotTo(HaveOccurred())
			// 创建时一次，本次一次，都使用已保存的 Token
			Expect(fake.receivedTokens()).To(Equal([]string{validToken, validToken}))
		})

		It("should test without token instead of the saved one when token is explicitly empty", func() {
			p := mustCreate("tse")
			fake.consoleOpen = true

			report, err := manager.TestConnectivity(ctx, workspaceID, accesspoint.ConnectivityParams{
				ID: p.ID.Hex(), ClientHosts: []string{"10.0.0.2"}, Token: lo.ToPtr(""),
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(report.OK).To(BeTrue())
			Expect(report.Message).To(ContainSubstring("token is empty"))
			// 只有创建时校验过 Token，本次没有带 Token
			Expect(fake.receivedTokens()).To(Equal([]string{validToken}))
			Expect(mustGet(p.ID).Token).To(Equal(validToken))
		})

		It("should reject an access point of another workspace", func() {
			p := mustCreate("tse")

			_, err := manager.TestConnectivity(ctx, "ws-other", accesspoint.ConnectivityParams{
				ID: p.ID.Hex(), ClientHosts: []string{"10.0.0.1"},
			})

			Expect(err).To(MatchError(accesspoint.ErrAccessPointNotFound))
		})
	})

	Describe("Create", func() {
		It("should save a connected access point with normalized fields", func() {
			p := mustCreate(" tse ")

			got := mustGet(p.ID)
			Expect(got.WorkspaceID).To(Equal(workspaceID))
			Expect(got.Name).To(Equal("tse"))
			Expect(got.ConsoleURL).To(Equal("http://console.example.com:8080"))
			Expect(got.ClientHosts).To(Equal([]string{"10.0.0.1", "polaris.example.com"}))
			Expect(got.Token).To(Equal(validToken))
			Expect(got.LastConnectivity.OK).To(BeTrue())
			Expect(got.Creator).To(Equal("ws-admin"))
			Expect(got.Updater).To(Equal("ws-admin"))
		})

		It("should write an audit record without the token", func() {
			p := mustCreate("tse")

			Eventually(recordedAudits).Should(HaveLen(1))
			r := recordedAudits()[0]
			Expect(r.OperationType).To(Equal(audit.OperationTypeCreate))
			Expect(r.ResourceType).To(Equal(audit.ResourceTypeWorkspace))
			Expect(r.ResourceID).To(Equal(workspaceID))
			Expect(r.Attribute).To(Equal(audit.AttributePolarisAccessPoint))
			Expect(string(r.Data.After)).To(MatchJSON(fmt.Sprintf(`{
				"id": %q, "name": "tse", "consoleURL": "http://console.example.com:8080",
				"clientHosts": ["10.0.0.1", "polaris.example.com"], "hasToken": true
			}`, p.ID.Hex())))
		})

		It("should allow an empty token", func() {
			params := createParams("tse")
			params.Token = ""

			p, err := manager.Create(ctx, workspaceID, params)

			Expect(err).NotTo(HaveOccurred())
			Expect(mustGet(p.ID).Token).To(BeEmpty())
		})

		It("should reject a duplicate name in the workspace", func() {
			mustCreate("tse")

			_, err := manager.Create(ctx, workspaceID, createParams("tse"))

			Expect(err).To(MatchError(accesspoint.ErrAccessPointNameExists))
			apiErr := accesspoint.APIError(err, "create").(*bkerrs.Error)
			Expect(apiErr.Code()).To(Equal(bkerrs.ErrCodeAlreadyExists))
		})

		It("should reject with INVALID_REQUEST when polaris is unreachable", func() {
			polarisURL = deadURL

			_, err := manager.Create(ctx, workspaceID, createParams("tse"))

			Expect(err).To(MatchError(accesspoint.ErrConnectivityFailed))
			apiErr := accesspoint.APIError(err, "create").(*bkerrs.Error)
			Expect(apiErr.Code()).To(Equal(bkerrs.ErrCodeInvalidRequest))
			Expect(apiErr.Error()).To(ContainSubstring("reason networkUnreachable"))
			Expect(apiErr.Details()).To(BeEmpty())
			Expect(apStore.Count(ctx, accesspoint.InWorkspace(workspaceID))).To(BeZero())
		})

		It("should reject an invalid token without leaking it", func() {
			params := createParams("tse")
			params.Token = badToken

			_, err := manager.Create(ctx, workspaceID, params)

			Expect(err).To(MatchError(accesspoint.ErrConnectivityFailed))
			Expect(err.Error()).To(ContainSubstring("reason tokenInvalid"))
			Expect(err.Error()).NotTo(ContainSubstring(badToken))
		})

		DescribeTable("should reject invalid arguments",
			func(modify func(*accesspoint.CreateParams)) {
				params := createParams("tse")
				modify(&params)

				_, err := manager.Create(ctx, workspaceID, params)

				Expect(err).To(MatchError(accesspoint.ErrAccessPointInvalid))
				apiErr := accesspoint.APIError(err, "create").(*bkerrs.Error)
				Expect(apiErr.Code()).To(Equal(bkerrs.ErrCodeInvalidArgument))
			},
			Entry("blank name", func(p *accesspoint.CreateParams) { p.Name = "  " }),
			Entry("blank console url", func(p *accesspoint.CreateParams) { p.ConsoleURL = "" }),
			Entry("unsupported console url scheme", func(p *accesspoint.CreateParams) {
				p.ConsoleURL = "ftp://console.example.com"
			}),
			Entry("no client host", func(p *accesspoint.CreateParams) { p.ClientHosts = []string{" "} }),
			Entry("client host with port", func(p *accesspoint.CreateParams) {
				p.ClientHosts = []string{"10.0.0.1:8090"}
			}),
			Entry("client host with scheme", func(p *accesspoint.CreateParams) {
				p.ClientHosts = []string{"http://10.0.0.1"}
			}),
			Entry("too many client hosts", func(p *accesspoint.CreateParams) {
				p.ClientHosts = nil
				for i := range accesspoint.MaxClientHosts + 1 {
					p.ClientHosts = append(p.ClientHosts, fmt.Sprintf("10.0.0.%d", i+1))
				}
			}),
		)
	})

	Describe("Update", func() {
		var p *accesspoint.PolarisAccessPoint

		BeforeEach(func() {
			p = mustCreate("tse")
		})

		It("should change addresses of an access point used by polaris configs when the new address connects", func() {
			configStore := testutil.NewStore(GinkgoT(), polaris.ConfigMeta)
			appID := "app-" + stringx.Random(6)
			Expect(configStore.Create(ctx, &polaris.PolarisConfig{
				AppID: appID, Name: "cfg", AccessPointID: p.ID.Hex(),
			})).To(Succeed())
			DeferCleanup(func() { _, _ = configStore.DeleteMany(ctx, polaris.ByApp(appID)) })

			got, err := manager.Update(ctx, workspaceID, p.ID.Hex(), accesspoint.UpdateParams{
				ClientHosts: &[]string{"10.0.0.9"},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(got.ClientHosts).To(Equal([]string{"10.0.0.9"}))
			Expect(mustGet(p.ID).ClientHosts).To(Equal([]string{"10.0.0.9"}))
		})

		It("should reject changing client hosts when the new address is unreachable", func() {
			polarisURL = deadURL

			_, err := manager.Update(ctx, workspaceID, p.ID.Hex(), accesspoint.UpdateParams{
				ClientHosts: &[]string{"10.0.0.9"},
			})

			Expect(err).To(MatchError(accesspoint.ErrConnectivityFailed))
			Expect(err.Error()).To(ContainSubstring("reason networkUnreachable"))
			Expect(mustGet(p.ID).ClientHosts).To(Equal(p.ClientHosts))
		})

		It("should not rerun the connectivity test when only name and console url change", func() {
			polarisURL = deadURL

			got, err := manager.Update(ctx, workspaceID, p.ID.Hex(), accesspoint.UpdateParams{
				Name: lo.ToPtr(" tse-2 "), ConsoleURL: lo.ToPtr("https://console-2.example.com/"),
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(got.Name).To(Equal("tse-2"))
			stored := mustGet(p.ID)
			Expect(stored.Name).To(Equal("tse-2"))
			Expect(stored.ConsoleURL).To(Equal("https://console-2.example.com"))
			Expect(stored.Token).To(Equal(validToken))
		})

		It("should keep the token when it is omitted and clear it when it is empty", func() {
			_, err := manager.Update(ctx, workspaceID, p.ID.Hex(), accesspoint.UpdateParams{Name: lo.ToPtr("tse-2")})
			Expect(err).NotTo(HaveOccurred())
			Expect(mustGet(p.ID).Token).To(Equal(validToken))

			_, err = manager.Update(ctx, workspaceID, p.ID.Hex(), accesspoint.UpdateParams{Token: lo.ToPtr("")})
			Expect(err).NotTo(HaveOccurred())
			Expect(mustGet(p.ID).Token).To(BeEmpty())
		})

		It("should reject an invalid new token", func() {
			_, err := manager.Update(ctx, workspaceID, p.ID.Hex(), accesspoint.UpdateParams{Token: lo.ToPtr(badToken)})

			Expect(err).To(MatchError(accesspoint.ErrConnectivityFailed))
			Expect(mustGet(p.ID).Token).To(Equal(validToken))
		})

		It("should reject renaming to an existing name", func() {
			mustCreate("tse-2")

			_, err := manager.Update(ctx, workspaceID, p.ID.Hex(), accesspoint.UpdateParams{Name: lo.ToPtr("tse-2")})

			Expect(err).To(MatchError(accesspoint.ErrAccessPointNameExists))
		})

		It("should write an audit record without the token", func() {
			_, err := manager.Update(ctx, workspaceID, p.ID.Hex(), accesspoint.UpdateParams{Token: lo.ToPtr("")})
			Expect(err).NotTo(HaveOccurred())

			Eventually(recordedAudits).Should(HaveLen(2))
			r := recordedAudits()[1]
			Expect(r.OperationType).To(Equal(audit.OperationTypeUpdate))
			Expect(string(r.Data.Before)).To(ContainSubstring(`"hasToken":true`))
			Expect(string(r.Data.After)).To(ContainSubstring(`"hasToken":false`))
			Expect(string(r.Data.Before) + string(r.Data.After)).NotTo(ContainSubstring(validToken))
		})

		It("should not update the platform access point", func() {
			_, err := manager.Update(
				ctx, workspaceID, accesspoint.PlatformID, accesspoint.UpdateParams{Name: lo.ToPtr("x")},
			)

			Expect(err).To(MatchError(accesspoint.ErrAccessPointNotFound))
		})
	})

	Describe("Delete", func() {
		var (
			p           *accesspoint.PolarisAccessPoint
			configStore *polaris.ConfigStore
			appID       string
		)

		BeforeEach(func() {
			p = mustCreate("tse")
			configStore = testutil.NewStore(GinkgoT(), polaris.ConfigMeta)
			appID = "app-" + stringx.Random(6)
			DeferCleanup(func() { _, _ = configStore.DeleteMany(ctx, polaris.ByApp(appID)) })
		})

		deleteAP := func() error {
			return manager.Delete(ctx, workspaceID, p.ID.Hex(), polaris.ListConfigUsagesByAccessPoint)
		}

		It("should delete an unused access point", func() {
			Expect(deleteAP()).To(Succeed())

			_, err := apStore.Get(ctx, store.ByID(p.ID))
			Expect(err).To(MatchError(accesspoint.ErrAccessPointNotFound))
			Eventually(recordedAudits).Should(HaveLen(2))
			Expect(recordedAudits()[1].OperationType).To(Equal(audit.OperationTypeDelete))
		})

		It("should reject when clusters are still bound", func() {
			Expect(bindStore.Create(ctx, &accesspoint.PolarisClusterBinding{
				WorkspaceID: workspaceID, ClusterID: "BCS-K8S-00002", AccessPointID: p.ID.Hex(),
			})).To(Succeed())
			Expect(bindStore.Create(ctx, &accesspoint.PolarisClusterBinding{
				WorkspaceID: workspaceID, ClusterID: "BCS-K8S-00001", AccessPointID: p.ID.Hex(),
			})).To(Succeed())

			err := deleteAP()

			Expect(err).To(MatchError(accesspoint.ErrAccessPointInUse))
			apiErr := accesspoint.APIError(err, "delete").(*bkerrs.Error)
			Expect(apiErr.Code()).To(Equal(bkerrs.ErrCodeInvalidRequest))
			Expect(apiErr.Details()).To(Equal(bkerrs.Details{bkerrs.NewDetail(
				bkerrs.ErrDetailCodePolarisAccessPointInUse,
				apiErr.Error(),
				bkerrs.WithSystem(bkerrs.SystemName),
				bkerrs.WithModule("polaris"),
				bkerrs.WithExtras(map[string]string{
					"clusterIDs": `["BCS-K8S-00001","BCS-K8S-00002"]`,
					"configs":    `[]`,
				}),
			)}))
			mustGet(p.ID)
		})

		It("should reject when a polaris config without scope envs still uses it", func() {
			Expect(configStore.Create(ctx, &polaris.PolarisConfig{
				AppID: appID, Name: "cfg", AccessPointID: p.ID.Hex(),
			})).To(Succeed())

			err := deleteAP()

			var inUse *accesspoint.AccessPointInUseError
			Expect(errors.As(err, &inUse)).To(BeTrue())
			Expect(inUse.ClusterIDs).To(BeEmpty())
			Expect(inUse.Extras()).To(Equal(map[string]string{
				"clusterIDs": `[]`,
				"configs":    fmt.Sprintf(`[{"appID":%q,"configName":"cfg"}]`, appID),
			}))
			mustGet(p.ID)
		})
	})

	Describe("List", func() {
		It("should list only workspace access points with bound cluster counts", func() {
			first := mustCreate("tse-1")
			second := mustCreate("tse-2")
			for i, apID := range []string{first.ID.Hex(), first.ID.Hex()} {
				Expect(bindStore.Create(ctx, &accesspoint.PolarisClusterBinding{
					WorkspaceID: workspaceID, ClusterID: fmt.Sprintf("BCS-K8S-%05d", i), AccessPointID: apID,
				})).To(Succeed())
			}
			otherWS := &accesspoint.PolarisAccessPoint{WorkspaceID: "ws-other", Name: "tse-1"}
			Expect(apStore.Create(ctx, otherWS)).To(Succeed())

			result, err := manager.List(ctx, workspaceID)

			Expect(err).NotTo(HaveOccurred())
			names := []string{}
			for _, ap := range result.AccessPoints {
				names = append(names, ap.Name)
			}
			Expect(names).To(Equal([]string{"tse-1", "tse-2"}))
			Expect(result.BoundClusterCounts).To(Equal(map[string]int{first.ID.Hex(): 2}))
			Expect(result.BoundClusterCounts[second.ID.Hex()]).To(BeZero())
		})

		It("should return an empty list when the workspace has no access point", func() {
			result, err := manager.List(ctx, workspaceID)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.AccessPoints).To(BeEmpty())
			Expect(result.BoundClusterCounts).To(BeEmpty())
		})
	})

	Describe("ListNamespaces", func() {
		It("should return not found for the platform access point", func() {
			_, err := manager.ListNamespaces(ctx, workspaceID, accesspoint.PlatformID)

			Expect(err).To(MatchError(accesspoint.ErrAccessPointNotFound))
		})

		It("should read live namespaces without the system namespace for a workspace access point", func() {
			p := mustCreate("tse")
			fake.namespaces = []string{"Polaris", "biz-a", "biz-b"}

			namespaces, err := manager.ListNamespaces(ctx, workspaceID, p.ID.Hex())

			Expect(err).NotTo(HaveOccurred())
			Expect(namespaces).To(Equal([]string{"biz-a", "biz-b"}))
		})

		It("should return ErrAccessPointUnreachable when polaris is unreachable", func() {
			p := mustCreate("tse")
			polarisURL = deadURL

			_, err := manager.ListNamespaces(ctx, workspaceID, p.ID.Hex())

			Expect(err).To(MatchError(accesspoint.ErrAccessPointUnreachable))
			apiErr := accesspoint.APIError(err, "list namespaces").(*bkerrs.Error)
			Expect(apiErr.Code()).To(Equal(bkerrs.ErrCodeInvalidRequest))
		})

		It("should return not found for an access point of another workspace", func() {
			p := mustCreate("tse")

			_, err := manager.ListNamespaces(ctx, "ws-other", p.ID.Hex())

			Expect(err).To(MatchError(accesspoint.ErrAccessPointNotFound))
		})
	})
})
