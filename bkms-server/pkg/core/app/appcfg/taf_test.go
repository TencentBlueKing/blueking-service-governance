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

package appcfg

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("mergeTAFContent", func() {
	Context("when working with TAF/TARS config files", func() {
		When("using simple overlay format", func() {
			It("should merge basic key=value pairs", func() {
				baseContent := `<tars>
    <application>
        <server>
            app=TestApp
            server=TestServer
            localip=127.0.0.1
        </server>
    </application>
</tars>`
				overlayContent := `<tars>
    <application>
        <server>
            localip=127.0.0.2
            port=8080
        </server>
    </application>
</tars>`
				result, err := mergeTAFContent(baseContent, &overlayContent)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(ContainSubstring("app=TestApp"))       // preserved
				Expect(result).To(ContainSubstring("server=TestServer")) // preserved
				Expect(result).To(ContainSubstring("localip=127.0.0.2")) // overridden
				Expect(result).To(ContainSubstring("port=8080"))         // added
			})

			It("should handle nested sections", func() {
				baseContent := `<tars>
    <application>
        <server>
            app=TestApp
            <TestApp.TestServer.HelloObj>
                endpoint=tcp -h 127.0.0.1 -p 10002
                maxconns=100000
            </TestApp.TestServer.HelloObj>
        </server>
    </application>
</tars>`
				overlayContent := `<tars>
    <application>
        <server>
            <TestApp.TestServer.HelloObj>
                maxconns=200000
                protocol=tars
            </TestApp.TestServer.HelloObj>
        </server>
    </application>
</tars>`
				result, err := mergeTAFContent(baseContent, &overlayContent)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(ContainSubstring("app=TestApp"))                        // preserved
				Expect(result).To(ContainSubstring("endpoint=tcp -h 127.0.0.1 -p 10002")) // preserved
				Expect(result).To(ContainSubstring("maxconns=200000"))                    // overridden
				Expect(result).To(ContainSubstring("protocol=tars"))                      // added
			})

			It("should add new sections from overlay", func() {
				baseContent := `<tars>
    <application>
        <server>
            app=TestApp
        </server>
    </application>
</tars>`
				overlayContent := `<tars>
    <application>
        <client>
            locator=tars.tarsregistry.QueryObj
        </client>
    </application>
</tars>`
				result, err := mergeTAFContent(baseContent, &overlayContent)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(ContainSubstring("app=TestApp"))                        // preserved
				Expect(result).To(ContainSubstring("<client>"))                           // added section
				Expect(result).To(ContainSubstring("locator=tars.tarsregistry.QueryObj")) // added
				Expect(result).To(ContainSubstring("</client>"))                          // added section end
			})
		})

		When("handling edge cases", func() {
			It("should return base when overlay is nil", func() {
				baseContent := `<tars>
    <application>
        app=Test
    </application>
</tars>`
				result, err := mergeTAFContent(baseContent, nil)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(baseContent))
			})

			It("should return base when overlay is empty string", func() {
				baseContent := `<tars>
    <application>
        app=Test
    </application>
</tars>`
				overlayContent := ""
				result, err := mergeTAFContent(baseContent, &overlayContent)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(baseContent))
			})

			It("should handle malformed TAF gracefully in base", func() {
				// Note: TarsGo conf package is lenient with malformed TAF,
				// so this test verifies it doesn't panic rather than expecting an error
				baseContent := `<tars><unclosed>`
				overlayContent := `<tars></tars>`
				// The function should either succeed or return an error, but not panic
				_, _ = mergeTAFContent(baseContent, &overlayContent)
			})

			It("should handle malformed TAF gracefully in overlay", func() {
				// Note: TarsGo conf package is lenient with malformed TAF,
				// so this test verifies it doesn't panic rather than expecting an error
				baseContent := `<tars></tars>`
				overlayContent := `<tars><unclosed>`
				// The function should either succeed or return an error, but not panic
				_, _ = mergeTAFContent(baseContent, &overlayContent)
			})
		})

		When("testing real TARS scenarios", func() {
			It("should handle typical TARS server config", func() {
				baseContent := `<tars>
    <application>
        enableset=n
        setdivision=NULL
        <server>
            node=tars.tarsnode.ServerObj@tcp -h 127.0.0.1 -p 19386 -t 60000
            app=TestApp
            server=HelloServer
            localip=127.0.0.1
            local=tcp -h 127.0.0.1 -p 8080 -t 3000
            modulename=TestApp.HelloServer
            <TestApp.HelloServer.HelloObj>
                allow
                endpoint=tcp -h 127.0.0.1 -p 10001 -t 60000
                maxconns=200000
                protocol=tars
                queuecap=10000
                queuetimeout=60000
                servant=TestApp.HelloServer.HelloObj
                threads=5
            </TestApp.HelloServer.HelloObj>
        </server>
        <client>
            locator=tars.tarsregistry.QueryObj@tcp -h 127.0.0.1 -p 17890
            sync-invoke-timeout=3000
            async-invoke-timeout=5000
            refresh-endpoint-interval=60000
        </client>
    </application>
</tars>`
				overlayContent := `<tars>
    <application>
        enableset=y
        setdivision=sz.test.1
        <server>
            localip=10.0.0.1
            <TestApp.HelloServer.HelloObj>
                endpoint=tcp -h 10.0.0.1 -p 10001 -t 60000
                threads=10
            </TestApp.HelloServer.HelloObj>
        </server>
    </application>
</tars>`

				// nolint:unused // Keep as reference for the expected merge result.
				// The Equal assertion is not used because TarsGo's GetDomain
				// iterates over a map, so child section order is non-deterministic.
				_ = `<tars>
    <application>
        enableset=y
        setdivision=sz.test.1
        <server>
            node=tars.tarsnode.ServerObj@tcp -h 127.0.0.1 -p 19386 -t 60000
            app=TestApp
            server=HelloServer
            localip=10.0.0.1
            local=tcp -h 127.0.0.1 -p 8080 -t 3000
            modulename=TestApp.HelloServer
            <TestApp.HelloServer.HelloObj>
                allow
                endpoint=tcp -h 10.0.0.1 -p 10001 -t 60000
                maxconns=200000
                protocol=tars
                queuecap=10000
                queuetimeout=60000
                servant=TestApp.HelloServer.HelloObj
                threads=10
            </TestApp.HelloServer.HelloObj>
        </server>
        <client>
            locator=tars.tarsregistry.QueryObj@tcp -h 127.0.0.1 -p 17890
            sync-invoke-timeout=3000
            async-invoke-timeout=5000
            refresh-endpoint-interval=60000
        </client>
    </application>
</tars>
`
				result, err := mergeTAFContent(baseContent, &overlayContent)

				Expect(err).NotTo(HaveOccurred())
				// Overridden values
				Expect(result).To(ContainSubstring("enableset=y"))
				Expect(result).To(ContainSubstring("setdivision=sz.test.1"))
				Expect(result).To(ContainSubstring("localip=10.0.0.1"))
				Expect(result).To(ContainSubstring("endpoint=tcp -h 10.0.0.1 -p 10001 -t 60000"))
				Expect(result).To(ContainSubstring("threads=10"))
				// Preserved values
				Expect(result).To(ContainSubstring("app=TestApp"))
				Expect(result).To(ContainSubstring("server=HelloServer"))
				Expect(result).To(ContainSubstring("maxconns=200000"))
				Expect(result).To(ContainSubstring("protocol=tars"))
				Expect(result).To(ContainSubstring("locator=tars.tarsregistry.QueryObj@tcp -h 127.0.0.1 -p 17890"))
			})
		})
	})
})
