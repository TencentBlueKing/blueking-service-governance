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

package hostport_test

import (
	"context"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.mongodb.org/mongo-driver/v2/mongo"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/addon/hostport"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

var _ = Describe("InjectFromStore", func() {
	const mainContainerName = "main"

	var (
		ctx       context.Context
		db        *mongo.Database
		testAppID string
	)

	BeforeEach(func() {
		ctx = context.Background()
		db = testutil.NewDB(GinkgoT())
		testAppID = "hostport-inject-" + stringx.Random(6)
		DeferCleanup(func() {
			_ = store.Of(db, hostport.HostPortConfigMeta).DeleteIfExists(ctx, hostport.ByApp(testAppID))
		})
	})

	It("merges webhook annotations and containerPorts", func() {
		_, err := hostport.ReplacePorts(ctx, db, testAppID, []int32{8080, 80})
		Expect(err).NotTo(HaveOccurred())

		meta := &metav1.ObjectMeta{Annotations: map[string]string{"keep": "1"}}
		containers := []corev1.Container{
			{Name: "sidecar"},
			{
				Name: mainContainerName,
				Ports: []corev1.ContainerPort{{
					Name:          "existing",
					ContainerPort: 80,
					Protocol:      corev1.ProtocolUDP,
				}},
			},
		}
		ports, err := hostport.InjectFromStore(ctx, db, testAppID, meta, containers, mainContainerName)
		Expect(err).NotTo(HaveOccurred())
		Expect(ports).To(Equal([]int32{80, 8080}))
		Expect(meta.Annotations).To(Equal(map[string]string{
			"keep":                        "1",
			hostport.AnnotationEnabledKey: hostport.AnnotationEnabledVal,
			hostport.AnnotationPortsKey:   "80,8080",
		}))
		Expect(containers[0].Ports).To(BeEmpty())
		Expect(containers[1].Ports).To(Equal([]corev1.ContainerPort{
			{Name: "existing", ContainerPort: 80, Protocol: corev1.ProtocolUDP},
			{ContainerPort: 8080},
		}))
	})

	It("writes nothing when there are no declared ports", func() {
		meta := &metav1.ObjectMeta{}
		containers := []corev1.Container{{Name: mainContainerName}}
		ports, err := hostport.InjectFromStore(ctx, db, testAppID, meta, containers, mainContainerName)
		Expect(err).NotTo(HaveOccurred())
		Expect(ports).To(Equal([]int32{}))
		Expect(meta.Annotations).To(BeNil())
		Expect(containers[0].Ports).To(BeEmpty())
	})
})
