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

var _ = Describe("Translate", func() {
	zh := WithLang(context.Background(), LangZH)
	en := WithLang(context.Background(), LangEN)

	It("should return the translation in the context language", func() {
		msg, ok := Translate(zh, "resource not found")
		Expect(ok).To(BeTrue())
		Expect(msg).To(Equal("资源不存在"))
	})

	It("should return the source text for the source language when the msgid is translated", func() {
		msg, ok := Translate(en, "resource not found")
		Expect(ok).To(BeTrue())
		Expect(msg).To(Equal("resource not found"))
	})

	It("should report false for msgids without any translation", func() {
		for _, ctx := range []context.Context{zh, en} {
			msg, ok := Translate(ctx, "no such message")
			Expect(ok).To(BeFalse())
			Expect(msg).To(Equal("no such message"))
		}
	})
})

var _ = Describe("T", func() {
	It("should translate and format the message", func() {
		ctx := WithLang(context.Background(), LangZH)
		Expect(T(ctx, "environment has %d deployed apps, cannot delete", 2)).
			To(Equal("环境已部署 2 个应用，无法删除"))
	})

	It("should format the source text when no translation exists", func() {
		ctx := WithLang(context.Background(), LangZH)
		Expect(T(ctx, "%d apps", 2)).To(Equal("2 apps"))
	})
})

var _ = Describe("load", func() {
	po := func(body string) fstest.MapFS {
		return fstest.MapFS{"locales/zh-cn.po": {Data: []byte("msgid \"\"\nmsgstr \"\"\n\n" + body)}}
	}

	It("should skip untranslated and fuzzy entries but still treat them as user-facing in the source language",
		func() {
			c, err := load(
				po("msgid \"a\"\nmsgstr \"\"\n\n#, fuzzy\nmsgid \"b\"\nmsgstr \"乙\"\n\nmsgid \"c\"\nmsgstr \"丙\"\n"),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.entries[LangZH]).To(Equal(map[string]string{"c": "丙"}))

			for _, msgid := range []string{"a", "b", "c"} {
				msg, ok := c.translate(LangEN, msgid)
				Expect(ok).To(BeTrue(), msgid)
				Expect(msg).To(Equal(msgid))
			}
			_, ok := c.translate(LangZH, "a")
			Expect(ok).To(BeFalse())
			_, ok = c.translate(LangEN, "not in po")
			Expect(ok).To(BeFalse())
		})

	It("should accept reordered and repeated explicit argument indexes", func() {
		_, err := load(po("msgid \"app %s in env %s\"\nmsgstr \"环境 %[2]s 中的应用 %[1]s（%[1]s）\"\n"))
		Expect(err).NotTo(HaveOccurred())
	})

	DescribeTable("should reject invalid locale files",
		func(files fstest.MapFS, errSubstring string) {
			_, err := load(files)
			Expect(err).To(MatchError(ContainSubstring(errSubstring)))
		},
		Entry("missing locale file", fstest.MapFS{}, "read locale file locales/zh-cn.po"),
		Entry("placeholder mismatch", po("msgid \"%d apps\"\nmsgstr \"%s 个应用\"\n"), "do not match"),
		Entry("missing placeholder", po("msgid \"app %s\"\nmsgstr \"应用\"\n"), "do not match"),
		Entry("duplicated msgid", po("msgid \"a\"\nmsgstr \"甲\"\n\nmsgid \"a\"\nmsgstr \"乙\"\n"), "duplicated"),
		Entry("same argument with conflicting verbs",
			po("msgid \"name %s\"\nmsgstr \"编号 %[1]d，名称 %[1]s\"\n"), "referenced by both"),
	)
})

var _ = Describe("placeholders", func() {
	DescribeTable("should map argument indexes to verbs",
		func(format string, expected map[int]rune) {
			Expect(placeholders(format)).To(Equal(expected))
		},
		Entry("plain verbs", "%s has %d items", map[int]rune{1: 's', 2: 'd'}),
		Entry("escaped percent", "100%% of %s", map[int]rune{1: 's'}),
		Entry("flags width precision", "%-8s %+.2f", map[int]rune{1: 's', 2: 'f'}),
		Entry("explicit indexes", "%[2]s then %[1]q", map[int]rune{2: 's', 1: 'q'}),
		Entry("star width", "%*d", map[int]rune{1: '*', 2: 'd'}),
		Entry("repeated index with the same verb", "%[1]s or %[1]s", map[int]rune{1: 's'}),
	)

	DescribeTable("should reject an argument referenced by different verbs",
		func(format string) {
			_, err := placeholders(format)
			Expect(err).To(MatchError(ContainSubstring("referenced by both")))
		},
		Entry("verb then verb", "%[1]d %[1]s"),
		Entry("verb order reversed", "%[1]s %[1]d"),
		Entry("star width and verb", "%[1]*[1]d"),
	)

	DescribeTable("should reject malformed formats",
		func(format, errSubstring string) {
			_, err := placeholders(format)
			Expect(err).To(MatchError(ContainSubstring(errSubstring)))
		},
		Entry("missing verb", "%s and %", "missing verb"),
		Entry("missing verb after flags and width", "%-8", "missing verb"),
		Entry("unclosed index", "%[1s", "unclosed argument index"),
		Entry("non-numeric index", "%[a]s", "invalid argument index"),
		Entry("zero index", "%[0]s", "invalid argument index"),
		Entry("invalid index in width", "%[x]*d", "invalid argument index"),
		Entry("invalid index in precision", "%.[x]*d", "invalid argument index"),
		Entry("invalid index in precision digits", "%.2[x]d", "invalid argument index"),
	)

	It("should report which side of the entry is malformed", func() {
		Expect(checkPlaceholders("%[1s", "%s")).To(MatchError(ContainSubstring("parse msgid")))
		Expect(checkPlaceholders("%s", "%[1s")).To(MatchError(ContainSubstring("parse msgstr")))
	})
})
