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

var _ = Describe("mergeYAMLContent", func() {
	Context("when working with Helm charts values", func() {
		When("using simple overlay format (version 1)", func() {
			It("should merge deployment values successfully", func() {
				baseValues := `
replicaCount: 1
image:
  repository: nginx
  tag: "1.20"
service:
  type: ClusterIP
  port: 80
resources:
  limits:
    cpu: 100m
    memory: 128Mi
`
				overlayValues := `
replicaCount: 3
image:
  tag: "1.21"
resources:
  limits:
    cpu: 200m
    memory: 256Mi
  requests:
    cpu: 100m
    memory: 128Mi
`
				result, err := mergeYAMLContent(baseValues, &overlayValues)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(ContainSubstring("replicaCount: 3"))
				Expect(result).To(ContainSubstring("tag: \"1.21\""))
				Expect(result).To(ContainSubstring("cpu: 200m"))
				Expect(result).To(ContainSubstring("memory: 256Mi"))
				Expect(result).To(ContainSubstring("cpu: 100m"))
				Expect(result).To(ContainSubstring("memory: 128Mi"))
				Expect(result).To(ContainSubstring("repository: nginx"))
				Expect(result).To(ContainSubstring("type: ClusterIP"))
				Expect(result).To(ContainSubstring("port: 80"))
			})

			It("should handle ingress configuration overlay", func() {
				baseValues := `
ingress:
  enabled: false
  className: ""
  annotations: {}
  hosts:
    - host: chart-example.local
      paths:
        - path: /
          pathType: Prefix
`
				overlayValues := `
ingress:
  enabled: true
  className: nginx
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
    cert-manager.io/cluster-issuer: letsencrypt-prod
  tls:
    - secretName: chart-example-tls
      hosts:
        - chart-example.local
`
				result, err := mergeYAMLContent(baseValues, &overlayValues)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(ContainSubstring("enabled: true"))
				Expect(result).To(ContainSubstring("className: \"nginx\""))
				Expect(result).To(ContainSubstring("nginx.ingress.kubernetes.io/rewrite-target: /"))
				Expect(result).To(ContainSubstring("cert-manager.io/cluster-issuer: letsencrypt-prod"))
				Expect(result).To(ContainSubstring("secretName: chart-example-tls"))
				// Original host configuration should be preserved
				Expect(result).To(ContainSubstring("chart-example.local"))
			})

			It("should merge complex application configuration", func() {
				baseValues := `
config:
  appName: myapp
  logLevel: INFO
  database:
    host: localhost
    port: 5432
  features:
    - authentication
    - logging
env:
  - name: NODE_ENV
    value: production
`
				overlayValues := `
config:
  logLevel: DEBUG
  database:
    host: db.example.com
    password: secret123
  features:
    - monitoring
    - caching
env:
  - name: API_URL
    value: https://api.example.com
  - name: TIMEOUT
    value: "30"
`
				result, err := mergeYAMLContent(baseValues, &overlayValues)

				Expect(err).NotTo(HaveOccurred())
				// preserved
				Expect(result).To(ContainSubstring("appName: myapp"))
				// overridden
				Expect(result).To(ContainSubstring("logLevel: DEBUG"))
				// overridden
				Expect(result).To(ContainSubstring("host: db.example.com"))
				// preserved
				Expect(result).To(ContainSubstring("port: 5432"))
				// added
				Expect(result).To(ContainSubstring("password: secret123"))
				// in features array
				Expect(result).To(ContainSubstring("monitoring"))
				// in features array
				Expect(result).To(ContainSubstring("caching"))
				Expect(result).To(ContainSubstring("name: API_URL"))
				Expect(result).To(ContainSubstring("name: TIMEOUT"))
			})
		})

		When("using version 2 overlay format", func() {
			It("should apply multiple patches in sequence", func() {
				baseValues := `
deployment:
  replicas: 1
  image: nginx:1.20
service:
  port: 80
`
				overlayValues := `
overlayVersion: "2"
patches:
  - deployment:
      replicas: 3
  - service:
      port: 8080
  - deployment:
      image: nginx:1.21
      resources:
        requests:
          cpu: 100m
`
				result, err := mergeYAMLContent(baseValues, &overlayValues)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(ContainSubstring("replicas: 3"))
				Expect(result).To(ContainSubstring("port: 8080"))
				Expect(result).To(ContainSubstring("image: nginx:1.21"))
				Expect(result).To(ContainSubstring("cpu: 100m"))
			})

			It("should handle empty patches field gracefully", func() {
				baseValues := `
key: value
`
				overlayValues := `
overlayVersion: "2"
patches: []
`
				result, err := mergeYAMLContent(baseValues, &overlayValues)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(baseValues))
			})
		})

		When("handling edge cases", func() {
			It("should return base values when overlay is nil", func() {
				baseValues := `
key: value
replicaCount: 2
`
				result, err := mergeYAMLContent(baseValues, nil)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(baseValues))
			})

			It("should return base when overlay is empty string", func() {
				baseValues := `
key: value
`
				overlayValues := ""
				result, err := mergeYAMLContent(baseValues, &overlayValues)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(baseValues))
			})

			It("should return error for unsupported overlay version", func() {
				baseValues := `
key: value
`
				overlayValues := `
overlayVersion: "3"
key: newvalue
`
				_, err := mergeYAMLContent(baseValues, &overlayValues)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unsupported overlayVersion"))
			})

			It("should return error for invalid YAML in overlay", func() {
				baseValues := `
key: value
`
				overlayValues := `
invalid: yaml: content:
  - unclosed
`
				_, err := mergeYAMLContent(baseValues, &overlayValues)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("parsing the overlay YAML"))
			})

			It("should handle version 2 with missing patches field", func() {
				baseValues := `
key: value
`
				overlayValues := `
overlayVersion: "2"
someOtherField: data
`
				result, err := mergeYAMLContent(baseValues, &overlayValues)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(baseValues))
			})

			It("should handle version 2 with wrong patches type", func() {
				baseValues := `
key: value
`
				overlayValues := `
overlayVersion: "2"
patches: "not-an-array"
`
				result, err := mergeYAMLContent(baseValues, &overlayValues)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(baseValues))
			})
		})

		When("testing real Helm chart scenarios", func() {
			It("should handle WordPress Helm chart values", func() {
				baseValues := `
username: user
password: password
email: user@example.com
mariadb:
  auth:
    rootPassword: blueking
    database: wordpress
persistence:
  enabled: true
  size: 10Gi
`

				overlayValues := `
username: admin
email: admin@example.com
mariadb:
  auth:
    rootPassword: bluekingNo1
persistence:
  size: 20Gi
  storageClass: fast-ssd
service:
  type: LoadBalancer
`

				result, err := mergeYAMLContent(baseValues, &overlayValues)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(ContainSubstring("username: admin"))
				Expect(result).To(ContainSubstring("email: admin@example.com"))
				Expect(result).To(ContainSubstring("rootPassword: bluekingNo1"))
				Expect(result).To(ContainSubstring("database: wordpress"))
				Expect(result).To(ContainSubstring("size: 20Gi"))
				Expect(result).To(ContainSubstring("storageClass: fast-ssd"))
				Expect(result).To(ContainSubstring("type: LoadBalancer"))
			})

			It("should handle Prometheus Helm chart values", func() {
				baseValues := `
server:
  global:
    scrapeInterval: 15s
  retention: "10d"
alertmanager:
  enabled: true
  persistence:
    enabled: false
`

				overlayValues := `
server:
  global:
    scrapeInterval: 30s
    evaluationInterval: 30s
  retention: "30d"
  resources:
    limits:
      cpu: 200m
      memory: 512Mi
alertmanager:
  persistence:
    enabled: true
    size: 5Gi
`
				result, err := mergeYAMLContent(baseValues, &overlayValues)

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(ContainSubstring("scrapeInterval: 30s"))
				Expect(result).To(ContainSubstring("evaluationInterval: 30s"))
				Expect(result).To(ContainSubstring("retention: \"30d\""))
				Expect(result).To(ContainSubstring("cpu: 200m"))
				Expect(result).To(ContainSubstring("memory: 512Mi"))
				// alertmanager enabled preserved
				Expect(result).To(ContainSubstring("enabled: true"))
				// persistence enabled overridden
				Expect(result).To(ContainSubstring("enabled: true"))
				Expect(result).To(ContainSubstring("size: 5Gi"))
			})
		})
	})
})

var _ = Describe("mergeContent", func() {
	Context("when selecting merge function by format", func() {
		It("should use YAML merge for yaml format", func() {
			baseContent := `key: value`
			overlayContent := `key: newvalue`
			result, err := mergeContent(baseContent, &overlayContent, FileFormatYAML)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(ContainSubstring("key: newvalue"))
		})

		It("should use TAF merge for taf format", func() {
			baseContent := `<tars><app>key=value</app></tars>`
			overlayContent := `<tars><app>key=newvalue</app></tars>`
			result, err := mergeContent(baseContent, &overlayContent, FileFormatTAF)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(ContainSubstring("key=newvalue"))
		})
	})
})
