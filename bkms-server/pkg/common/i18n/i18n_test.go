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

package i18n

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing/fstest"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Negotiate", func() {
	DescribeTable("should pick the language by cookie first, then Accept-Language",
		func(cookie, acceptLanguage string, expected Lang) {
			Expect(Negotiate(cookie, acceptLanguage)).To(Equal(expected))
		},
		Entry("cookie zh-cn", "zh-cn", "en-US", LangZH),
		Entry("cookie en", "en", "zh-CN", LangEN),
		Entry("invalid cookie falls back to header", "!!", "en-GB,en;q=0.9", LangEN),
		Entry("header weights", "", "fr;q=1.0,en;q=0.8,zh;q=0.5", LangEN),
		Entry("traditional chinese", "", "zh-TW", LangZH),
		Entry("unsupported language", "", "fr", DefaultLang),
		Entry("nothing provided", "", "", DefaultLang),
	)
})

var _ = Describe("Middleware", func() {
	It("should store the negotiated language in request context", func() {
		gin.SetMode(gin.TestMode)
		var got Lang
		router := gin.New()
		router.Use(Middleware())
		router.GET("/ping", func(c *gin.Context) {
			got = LangFromContext(c.Request.Context())
		})

		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.AddCookie(&http.Cookie{Name: LangCookieName, Value: "en"})
		router.ServeHTTP(httptest.NewRecorder(), req)

		Expect(got).To(Equal(LangEN))
	})

	It("should use default language when context has no language", func() {
		Expect(LangFromContext(context.Background())).To(Equal(DefaultLang))
	})
})

var _ = Describe("TryT", func() {
	It("should render the embedded message in the context language", func() {
		params := map[string]string{"appCount": "2"}

		msg, ok := TryT(WithLang(context.Background(), LangZH), "CORE_ENV_HAS_APPS_CANNOT_UPDATE_CLUSTER", params)
		Expect(ok).To(BeTrue())
		Expect(msg).To(Equal("环境已部署 2 个应用，无法修改集群配置"))

		msg, ok = TryT(WithLang(context.Background(), LangEN), "CORE_ENV_HAS_APPS_CANNOT_UPDATE_CLUSTER", params)
		Expect(ok).To(BeTrue())
		Expect(msg).To(Equal("Environment has 2 deployed app(s) and cannot update cluster configuration"))
	})

	It("should report false when the key is unknown", func() {
		_, ok := TryT(context.Background(), "NO_SUCH_KEY", nil)
		Expect(ok).To(BeFalse())
		Expect(HasKey("NO_SUCH_KEY")).To(BeFalse())
	})

	It("should report false instead of rendering <no value> when a param is missing", func() {
		_, ok := TryT(context.Background(), "CORE_ENV_HAS_APPS_CANNOT_UPDATE_CLUSTER", nil)
		Expect(ok).To(BeFalse())
	})
})

var _ = Describe("T", func() {
	ctx := WithLang(context.Background(), LangZH)

	It("should render the message with optional params", func() {
		Expect(T(ctx, "NOT_FOUND")).To(Equal("资源不存在"))
		Expect(T(ctx, "CORE_ENV_HAS_APPS_CANNOT_UPDATE_CLUSTER", map[string]string{"appCount": "2"})).
			To(Equal("环境已部署 2 个应用，无法修改集群配置"))
	})

	It("should return the key when the message is unavailable", func() {
		Expect(T(ctx, "NO_SUCH_KEY")).To(Equal("NO_SUCH_KEY"))
		Expect(T(ctx, "CORE_ENV_HAS_APPS_CANNOT_UPDATE_CLUSTER")).To(Equal("CORE_ENV_HAS_APPS_CANNOT_UPDATE_CLUSTER"))
	})
})

var _ = Describe("load", func() {
	DescribeTable("should reject invalid locale files",
		func(files fstest.MapFS, errSubstring string) {
			_, err := load(files)
			Expect(err).To(MatchError(ContainSubstring(errSubstring)))
		},
		Entry("missing language", fstest.MapFS{
			"locales/a.yaml": {Data: []byte("FOO:\n  zh: \"中文\"\n")},
		}, "lacks en text"),
		Entry("bad template", fstest.MapFS{
			"locales/a.yaml": {Data: []byte("FOO:\n  zh: \"{{.x\"\n  en: \"x\"\n")},
		}, "parse template of locale key FOO"),
		Entry("duplicated key across files", fstest.MapFS{
			"locales/a.yaml": {Data: []byte("FOO:\n  zh: \"a\"\n  en: \"a\"\n")},
			"locales/b.yaml": {Data: []byte("FOO:\n  zh: \"b\"\n  en: \"b\"\n")},
		}, "locale key FOO is duplicated"),
	)
})
