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

package dbfactory

import (
	"context"

	"github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/component"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

// LoadBuiltinComponents loads builtin component definitions from files, it's useful for tests
// that depend on these components.
//
// Args:
// - path: the folder path where the component definitions are stored.
func LoadBuiltinComponents(ctx context.Context, client *mongo.Client, path string) {
	compDefStore, err := component.NewComponentDefStoreMongo(database.Client(), database.Name())
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	err = component.LoadBuiltinFromFolder(ctx, compDefStore, path)
	gomega.Expect(err).To(gomega.Not(gomega.HaveOccurred()))
}
