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

package env

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
)

var _ = Describe("Test EnvService", func() {
	var diApp *fxtest.App
	var envSvc *EnvService

	var err error
	var ctx context.Context
	var envStore model.EnvironmentStore

	BeforeEach(func() {
		ctx = context.Background()

		diApp = fxtest.New(
			GinkgoT(),
			FxModule,
			fx.Populate(
				&envSvc,
				&envStore,
			),
		)
		diApp.RequireStart()
	})

	AfterEach(func() {
		Expect(envStore.DeleteAll(ctx)).NotTo(HaveOccurred())
		diApp.RequireStop()
	})

	Context("test env service methods", func() {
		var envName string
		var workspaceID string
		var Cluster model.BizCluster
		var envID bson.ObjectID
		var envType string

		BeforeEach(func() {
			envName = stringx.Random(10)
			workspaceID = stringx.Random(10)
			envType = stringx.Random(10)
			Cluster = model.BizCluster{
				ClusterID:   stringx.Random(10),
				ClusterType: stringx.Random(10),
				Namespace:   stringx.Random(10),
			}
			// test: create environment
			envID, err = envSvc.Create(
				ctx,
				&model.Environment{
					Name:        envName,
					WorkspaceID: workspaceID,
					Cluster:     Cluster,
					Type:        envType,
					Description: stringx.Random(10),
				},
			)
			Expect(err).NotTo(HaveOccurred())
		})
		AfterEach(func() {
			// test: delete environment
			err = envSvc.Delete(ctx, envID)
		})

		It("test list environments", func() {
			environments, _ := envSvc.ListStdEnvs(ctx, workspaceID)
			Expect(environments).To(HaveLen(1))
			Expect(environments[0].Name).To(Equal(envName))
			Expect(environments[0].Status).To(Equal(model.EnvStatusReady))
		})

		It("test update cluster namespace failed when app already deployed", func() {
			appID := stringx.Random(10)
			Expect(envSvc.AddApp(ctx, envID, appID)).NotTo(HaveOccurred())

			newNamespace := stringx.Random(10)
			updateErr := envSvc.Update(ctx, envID, &model.EnvironmentUpdateData{
				Namespace: &newNamespace,
			})
			Expect(updateErr.Error()).To(ContainSubstring("cannot update cluster"))
		})
	})
})
