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

package migrate_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component/migrate"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

const (
	componentDefsCollection    = "component_defs"
	importPolarisComponentName = "ImportPolaris"
	legacyImportPolarisOutput  = `type: ComponentOutput
name: ImportPolaris
patcher:
  "spec.template.spec":
    containers:
      - name: "{{ .bkmsContainerName }}"
        env:
          - name: "{{ .instanceKey }}_polarisToken"
            value: {{ .polarisToken }}
          - name: "{{ .instanceKey }}_serviceport"
            value: "{{ .servicePort}}"
        ports:
          - containerPort: {{ .servicePort }}
            protocol: "TCP"
            name: "polaris-{{ .servicePort }}"
spec:
  - apiVersion: tkex.tencent.com/v1
    kind: PolarisConfig
    metadata:
      name: "{{ .name }}-polaris"
    spec:
      polaris:
        name: {{ .polarisName }}
        namespace: {{ .polarisNamespace }}
        token: {{ .polarisToken }}
      services:
        - name: "{{ .name }}-polaris-service"
          namespace: {{ .bkmsEnvNamespace }}
          port: {{ .servicePort }}
          direct: {{ .direct }}
          keepNotReadyPod: {{ .keepNotReadyPod }}
          enableHealthCheck: {{ .enableHealthCheck }}
          weight: {{.weight}}
          {{- if .serviceLabels }}
          extraMeta:
          {{- range $key, $value := .serviceLabels }}
            {{ $key }}: "{{ $value }}"
          {{- end }}
          {{- end }}
  - apiVersion: v1
    kind: Service
    metadata:
      name: "{{ .name }}-polaris-service"
    spec:
      selector:
        app.kubernetes.io/name: {{ .bkmsAppName }}
      ports:
        - protocol: TCP
          port: {{ .servicePort }}
          targetPort: {{ .servicePort }}`
)

var _ = Describe("Component patch migration", func() {
	var (
		ctx                    context.Context
		db                     *mongo.Database
		migrator               *migrate.Migrator
		insertLegacyComponents func()
	)

	BeforeEach(func() {
		ctx = context.Background()
		Expect(testutil.CleanupCollection(componentDefsCollection)).To(Succeed())
		db = database.Client().Database(database.Name())
		migrator = migrate.New(db)
		insertLegacyComponents = func() {
			_, err := db.Collection(componentDefsCollection).InsertMany(ctx, []any{
				bson.M{
					"name": "Demo", "version": component.DefaultComponentDefVersion,
					"properties": bson.A{}, "output": "patcher:\n  spec.replicas: 2\n",
				},
				bson.M{
					"name": importPolarisComponentName, "version": component.DefaultComponentDefVersion,
					"properties": bson.A{}, "output": legacyImportPolarisOutput, "isBuiltin": true,
				},
			})
			Expect(err).NotTo(HaveOccurred())
		}
	})

	It("reports all changes without writing during a dry run", func() {
		insertLegacyComponents()

		result, err := migrator.Run(ctx, true)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.DryRun).To(BeTrue())
		Expect(result.Summary.Migrated).To(Equal(2))
		Expect(result.Summary.Failed).To(BeZero())

		var unchanged bson.M
		Expect(db.Collection(componentDefsCollection).FindOne(
			ctx,
			bson.M{"name": "Demo"},
		).Decode(&unchanged)).To(Succeed())
		Expect(unchanged).To(HaveKey("output"))
		Expect(unchanged).NotTo(HaveKey("patchers"))
		count, err := db.Collection(componentDefsCollection).CountDocuments(ctx, bson.M{
			"name": importPolarisComponentName,
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(count).To(Equal(int64(1)))
	})

	It("applies all valid changes and skips migrated definitions on rerun", func() {
		insertLegacyComponents()
		collection := db.Collection(componentDefsCollection)

		result, err := migrator.Run(ctx, false)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Summary.Migrated).To(Equal(2))

		var migrated bson.M
		Expect(collection.FindOne(ctx, bson.M{"name": "Demo"}).Decode(&migrated)).To(Succeed())
		Expect(migrated).NotTo(HaveKey("output"))
		Expect(migrated["patchers"]).To(HaveLen(1))
		Expect(migrated["specs"]).To(HaveLen(0))
		var migratedImportPolaris component.ComponentDef
		Expect(collection.FindOne(
			ctx,
			bson.M{"name": importPolarisComponentName},
		).Decode(&migratedImportPolaris)).To(Succeed())
		Expect(migratedImportPolaris.Patchers).To(HaveLen(1))
		Expect(migratedImportPolaris.Specs).To(HaveLen(2))
		Expect(migratedImportPolaris.Specs[0]).To(ContainSubstring("{{- if .serviceLabels }}"))

		store, err := component.NewComponentDefStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
		Expect(component.LoadBuiltinFromFolder(
			ctx,
			store,
			"../assets/comps/ImportPolaris_v1.0.0.yaml",
		)).To(Succeed())
		loadedImportPolaris, err := store.Get(
			ctx,
			importPolarisComponentName,
			component.DefaultComponentDefVersion,
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(loadedImportPolaris.Patchers).To(Equal(migratedImportPolaris.Patchers))
		Expect(loadedImportPolaris.Specs).To(Equal(migratedImportPolaris.Specs))

		result, err = migrator.Run(ctx, false)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Summary.Migrated).To(BeZero())
		Expect(result.Summary.Skipped).To(Equal(2))
	})

	It("skips a one-sided definition persisted by the component store", func() {
		store, err := component.NewComponentDefStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
		Expect(store.Create(ctx, &component.ComponentDef{
			Name:     "PatchOnly",
			Version:  component.DefaultComponentDefVersion,
			Patchers: []string{"spec:\n  replicas: 1\n"},
		})).To(Succeed())

		result, err := migrator.Run(ctx, false)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Summary.Skipped).To(Equal(1))
	})

	It("stops on an invalid component after processing earlier definitions", func() {
		collection := db.Collection(componentDefsCollection)
		_, err := collection.InsertMany(ctx, []any{
			bson.M{
				"name": "Demo", "version": component.DefaultComponentDefVersion,
				"output": "patcher:\n  spec.replicas: 2\n",
			},
			bson.M{
				"name": "Empty", "version": component.DefaultComponentDefVersion,
				"output": "patcher: {}\n",
			},
		})
		Expect(err).NotTo(HaveOccurred())

		result, err := migrator.Run(ctx, false)
		Expect(err).To(MatchError(ContainSubstring("at least one patcher or spec")))
		Expect(result.Summary.Migrated).To(Equal(1))
		Expect(result.Summary.Failed).To(Equal(1))

		var migrated bson.M
		Expect(collection.FindOne(ctx, bson.M{"name": "Demo"}).Decode(&migrated)).To(Succeed())
		Expect(migrated).NotTo(HaveKey("output"))
		Expect(migrated).To(HaveKey("patchers"))
	})
})
