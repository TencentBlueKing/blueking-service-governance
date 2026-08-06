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

package secret

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/TencentBlueKing/gopkg/mapx"
	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/build/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/testutil"
	bkmsenv "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/cluster"
)

var _ = Describe("imagePullSecretSyncer", func() {
	var (
		ctx         context.Context
		mocker      *mockey.Mocker
		syncer      *ImagePullSecretSyncer
		workspaceID string
		clusterID   string
		namespace   string
		env         *bkmsenv.Environment
		clientSet   *kubernetes.Clientset

		secretName = "bkms-image-pull-secret-test-workspace" //nolint:gosec // G101: not real credential
		// 测试用的镜像仓库认证信息
		auths map[string]any
	)

	BeforeEach(func() {
		ctx = context.Background()
		// 工作空间 ID
		workspaceID = "test-workspace"
		// 集群 ID
		clusterID = "test-cluster"
		// 每个单测使用不同的随机命名空间
		namespace = stringx.Random(8)
		// 初始化环境信息
		env = &bkmsenv.Environment{
			WorkspaceID: workspaceID,
			Cluster: bkmsenv.BizCluster{
				ClusterID: clusterID,
				Namespace: namespace,
			},
		}
		// 默认的镜像仓库认证信息
		auths = map[string]any{
			"https://mirrors.example.com": map[string]any{
				"username": "admin",
				"password": "password",
			},
			"https://mirrors.blueking.com": map[string]any{
				"username": "blueking",
				"password": "blueking",
			},
		}
		syncer = NewImagePullSecretSyncer(env, "", nil)

		var err error
		cfg, err := testutil.TestClusterConfig("")
		if errors.Is(err, testutil.ErrKubeConfigNotFound) {
			Skip(err.Error())
		}
		Expect(err).NotTo(HaveOccurred())

		// Set up the mocker to make the manager always use the test config.
		mocker = mockey.Mock(cluster.NewConfig).Return(cfg).Build()

		clientSet, err = kubernetes.NewForConfig(cfg.Rest)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		mocker.Release()
	})

	Describe("Sync", func() {
		Context("when namespace exists", func() {
			BeforeEach(func() {
				// 预先初始化命名空间
				_, err := clientSet.CoreV1().Namespaces().Create(
					ctx, &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}, metav1.CreateOptions{},
				)
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				// 回收命名空间
				err := clientSet.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
			})

			Context("when secret does not exist", func() {
				It("should create the secret successfully", func() {
					mockey.PatchConvey("test", GinkgoT(), func() {
						mockey.Mock((*ImagePullSecretSyncer).genAuthsData).Return(auths, nil).Build()

						// 执行检查 - 创建流程
						err := syncer.Sync(ctx)
						Expect(err).NotTo(HaveOccurred())

						// 检查集群中确实已经存在该 imagePullSecret
						_, err = clientSet.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())
					})
				})
			})

			Context("when secret already exists", func() {
				It("should upsert the secret", func() {
					mockey.PatchConvey("test", GinkgoT(), func() {
						mockey.Mock((*ImagePullSecretSyncer).genAuthsData).Return(auths, nil).Build()

						// 先初始化 imagePullSecret 占位
						_, err := clientSet.CoreV1().Secrets(namespace).Create(
							ctx, &v1.Secret{
								ObjectMeta: metav1.ObjectMeta{Name: secretName},
								Type:       v1.SecretTypeDockerConfigJson,
								Data: map[string][]byte{
									".dockerconfigjson": []byte(
										`{"auths":{"https://mirrors.exp.com": {"password": "a", "username": "b"}}}`,
									),
								},
							}, metav1.CreateOptions{},
						)
						Expect(err).NotTo(HaveOccurred())

						err = syncer.Sync(ctx)
						Expect(err).NotTo(HaveOccurred())

						secret, err := clientSet.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
						Expect(err).NotTo(HaveOccurred())

						var dockerCfgJson map[string]any
						err = json.Unmarshal(secret.Data[".dockerconfigjson"], &dockerCfgJson)
						Expect(err).NotTo(HaveOccurred())

						Expect(dockerCfgJson).To(Equal(map[string]any{"auths": auths}))
					})
				})
			})
		})

		Context("when namespace does not exist", func() {
			It("should return an error", func() {
				err := syncer.Sync(ctx)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(fmt.Sprintf("get namespace: %s: k8s resource not found", namespace)))
			})
		})
	})

	Describe("genSecretManifest", func() {
		It("should generate a valid secret manifest", func() {
			manifest, err := syncer.genSecretManifest(auths)
			Expect(err).NotTo(HaveOccurred())

			secretType := mapx.GetStr(manifest, "type")
			Expect(secretType).To(Equal("kubernetes.io/dockerconfigjson"))

			encoded := mapx.GetStr(manifest, []string{"data", ".dockerconfigjson"})
			Expect(encoded).NotTo(Equal(""))

			data, err := base64.StdEncoding.DecodeString(encoded)
			Expect(err).NotTo(HaveOccurred())

			var dockerCfgJson map[string]any
			err = json.Unmarshal(data, &dockerCfgJson)
			Expect(err).NotTo(HaveOccurred())

			Expect(dockerCfgJson).To(Equal(map[string]any{"auths": auths}))
		})

		It("same map should generate same .dockerconfigjson", func() {
			// 相同 map 生成应该得到相同的结果（不受 map key 顺序影响）
			sameAuths := map[string]any{
				"https://mirrors.blueking.com": map[string]any{
					"password": "blueking",
					"username": "blueking",
				},
				"https://mirrors.example.com": map[string]any{
					"password": "password",
					"username": "admin",
				},
			}
			manifest1, err := syncer.genSecretManifest(auths)
			Expect(err).NotTo(HaveOccurred())

			manifest2, err := syncer.genSecretManifest(sameAuths)
			Expect(err).NotTo(HaveOccurred())

			paths := []string{"data", ".dockerconfigjson"}
			encoded1 := mapx.GetStr(manifest1, paths)
			encoded2 := mapx.GetStr(manifest2, paths)
			Expect(encoded1).To(Equal(encoded2))
		})
	})

	Describe("addImageConfigAuth", func() {
		It("should add auth from image build config", func() {
			addImageConfigAuth(auths, &build.Config{
				AppID: "app-a",
				Image: &build.ImageConfig{
					Name:     "registry.example.com/group/app-a",
					Username: "app-a-user",
					Password: "app-a-pass",
				},
			})

			Expect(auths).To(HaveKeyWithValue(
				"registry.example.com/group/app-a",
				map[string]any{"username": "app-a-user", "password": "app-a-pass"},
			))
		})

		It("should skip image config without credential", func() {
			addImageConfigAuth(auths, &build.Config{
				AppID: "app-b",
				Image: &build.ImageConfig{Name: "registry.example.com/group/app-b"},
			})

			Expect(auths).NotTo(HaveKey("registry.example.com/group/app-b"))
		})

		It("should overwrite workspace auth when keys conflict", func() {
			addImageConfigAuth(auths, &build.Config{
				AppID: "app-a",
				Image: &build.ImageConfig{
					Name:     "https://mirrors.example.com",
					Username: "app-a-user",
					Password: "app-a-pass",
				},
			})

			Expect(auths).To(HaveKeyWithValue(
				"https://mirrors.example.com",
				map[string]any{"username": "app-a-user", "password": "app-a-pass"},
			))
		})
	})
})
