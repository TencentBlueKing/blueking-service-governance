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

package appmodel

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8skind "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/kind"
)

// configMapKey 返回指定名称的 ConfigMap 资源引用
func configMapKey(name string) ResourceKey {
	return ResourceKey{Kind: k8skind.CM, Name: name}
}

var _ = Describe("expiredConfigMaps", func() {
	gdKey := ResourceKey{Kind: k8skind.GameDeploy, Name: "demo"}
	current := ResourceKeys{gdKey, configMapKey("cm-new")}
	newRecord := func(status Status, keys ...ResourceKey) Record {
		return Record{Status: status, ResourceKeys: append(ResourceKeys{gdKey}, keys...)}
	}

	It("should expire ConfigMaps older than the latest deployed record", func() {
		expired := getExpiredConfigMaps([]Record{
			newRecord(StatusDeploying, configMapKey("cm-c")),
			newRecord(StatusCanceled, configMapKey("cm-b")),
			newRecord(StatusDeployed, configMapKey("cm-a")),
			newRecord(StatusDeployed, configMapKey("cm-z")),
		}, current)
		Expect(expired).To(Equal(ResourceKeys{configMapKey("cm-z")}))
	})

	It("should keep rolled back records and the rollback target", func() {
		expired := getExpiredConfigMaps([]Record{
			newRecord(StatusRolledBack, configMapKey("cm-b")),
			newRecord(StatusDeployed, configMapKey("cm-a")),
			newRecord(StatusFailed, configMapKey("cm-y")),
		}, current)
		Expect(expired).To(Equal(ResourceKeys{configMapKey("cm-y")}))
	})

	It("should expire uninstalled records and anything older", func() {
		expired := getExpiredConfigMaps([]Record{
			newRecord(StatusDeploying, configMapKey("cm-b")),
			newRecord(StatusUninstalled, configMapKey("cm-a")),
			newRecord(StatusDeploying, configMapKey("cm-z")),
		}, current)
		Expect(expired).To(Equal(ResourceKeys{configMapKey("cm-a"), configMapKey("cm-z")}))
	})

	It("should expire nothing when no deployed record exists", func() {
		expired := getExpiredConfigMaps([]Record{
			newRecord(StatusFailed, configMapKey("cm-b")),
			newRecord(StatusDeploying, configMapKey("cm-a")),
		}, current)
		Expect(expired).To(BeEmpty())
	})

	It("should not expire a ConfigMap that is still in use under the same name", func() {
		svcKey := ResourceKey{Kind: k8skind.SVC, Name: "demo"}
		records := []Record{
			newRecord(StatusDeploying, configMapKey("cm-a"), svcKey),
			newRecord(StatusDeployed, configMapKey("cm-b")),
			newRecord(StatusDeployed, configMapKey("cm-a")),
			newRecord(StatusDeployed, configMapKey("cm-z")),
			newRecord(StatusDeployed, configMapKey("cm-z")),
		}
		Expect(getExpiredConfigMaps(records, current)).To(Equal(ResourceKeys{configMapKey("cm-z")}))
		// 本次部署的配置内容与 cm-z 相同
		Expect(getExpiredConfigMaps(records, ResourceKeys{gdKey, configMapKey("cm-z")})).To(BeEmpty())
	})

	It("should order records by creation time regardless of the input order", func() {
		now := time.Now()
		deployed := newRecord(StatusDeployed, configMapKey("cm-a"))
		deployed.CreatedAt = now.Add(-time.Minute)
		older := newRecord(StatusDeployed, configMapKey("cm-z"))
		older.CreatedAt = now.Add(-2 * time.Minute)
		latest := newRecord(StatusDeploying, configMapKey("cm-b"))
		latest.CreatedAt = now

		expired := getExpiredConfigMaps([]Record{older, latest, deployed}, current)
		Expect(expired).To(Equal(ResourceKeys{configMapKey("cm-z")}))
	})
})

var _ = Describe("referencedConfigMaps", func() {
	It("should return every ConfigMap once, newest first, ignoring other kinds", func() {
		gdKey := ResourceKey{Kind: k8skind.GameDeploy, Name: "demo"}
		configMaps := listRefConfigMaps([]Record{
			{Status: StatusUninstalling, ResourceKeys: ResourceKeys{gdKey, configMapKey("cm-b")}},
			{Status: StatusDeployed, ResourceKeys: ResourceKeys{gdKey, configMapKey("cm-a")}},
			{Status: StatusUninstalled, ResourceKeys: ResourceKeys{gdKey, configMapKey("cm-b")}},
		})
		Expect(configMaps).To(Equal(ResourceKeys{configMapKey("cm-b"), configMapKey("cm-a")}))
	})
})
