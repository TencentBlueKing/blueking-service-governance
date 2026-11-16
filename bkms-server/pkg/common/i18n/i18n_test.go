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

var _ = Describe("Localize", func() {
	It("should render the embedded message in the context language", func() {
		params := map[string]string{"itemName": "a"}

		msg, ok := Localize(WithLang(context.Background(), LangZH), "PORTPOOL_ITEM_NAME_DUPLICATED", params)
		Expect(ok).To(BeTrue())
		Expect(msg).To(Equal("端口池条目名称 a 重复"))

		msg, ok = Localize(WithLang(context.Background(), LangEN), "PORTPOOL_ITEM_NAME_DUPLICATED", params)
		Expect(ok).To(BeTrue())
		Expect(msg).To(Equal("Port pool item name a is duplicated"))
	})

	It("should report false when the key is unknown", func() {
		_, ok := Localize(context.Background(), "NO_SUCH_KEY", nil)
		Expect(ok).To(BeFalse())
		Expect(Has("NO_SUCH_KEY")).To(BeFalse())
	})

	It("should report false instead of rendering <no value> when a param is missing", func() {
		_, ok := Localize(context.Background(), "PORTPOOL_ITEM_NAME_DUPLICATED", nil)
		Expect(ok).To(BeFalse())
	})
})

var _ = Describe("LocalizeOr", func() {
	It("should return the localized message or the fallback", func() {
		ctx := WithLang(context.Background(), LangEN)
		Expect(LocalizeOr(ctx, "NOT_FOUND", nil, "fallback")).To(Equal("Resource not found"))
		Expect(LocalizeOr(ctx, "NO_SUCH_KEY", nil, "fallback")).To(Equal("fallback"))
	})
})

var _ = Describe("LocalizeStored", func() {
	ctx := WithLang(context.Background(), LangEN)
	params := map[string]string{"itemName": "a"}

	It("should translate a stored value equal to the chinese default", func() {
		Expect(LocalizeStored(ctx, "PORTPOOL_ITEM_NAME_DUPLICATED", params, "端口池条目名称 a 重复")).
			To(Equal("Port pool item name a is duplicated"))
	})

	It("should keep a value modified by the user", func() {
		Expect(LocalizeStored(ctx, "PORTPOOL_ITEM_NAME_DUPLICATED", params, "用户改过的名称")).
			To(Equal("用户改过的名称"))
	})

	It("should keep the stored value when the key is unknown", func() {
		Expect(LocalizeStored(ctx, "NO_SUCH_KEY", nil, "原值")).To(Equal("原值"))
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
