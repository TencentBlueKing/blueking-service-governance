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

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	"github.com/bytedance/mockey"
	"github.com/gin-gonic/gin"
	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
	infrasreg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/taskq"
	ginperm "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/perm"
	_ "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/ginutils/validators"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/customruntime"
	bkmsreg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/registry"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/image/snapshot"
)

var _ = Describe("Custom build image tags", func() {
	const (
		registryAddr = "docker.bkrepo.example.com/demo/repo"
		imageName    = registryAddr + "/my-golang"
	)

	var (
		ctx           context.Context
		store         customruntime.Store
		snapshotStore snapshot.SnapshotStore
		router        *gin.Engine
		workspaceID   string
		repoKey       string
		// 以下三者由各用例改写：permErr 非空时权限校验直接失败，另两个控制远程拉取结果
		permErr    error
		remoteTags []string
		remoteErr  error
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		ctx = context.Background()
		permErr = nil
		remoteTags, remoteErr = nil, nil
		workspaceID = "ws-tags-" + stringx.Random(6)
		repoKey = snapshot.GenerateRepoKey(imageName, "ws-user", "ws-pass")

		var err error
		store, err = customruntime.NewStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())
		snapshotStore, err = snapshot.NewSnapshotStoreMongo(database.Client(), database.Name())
		Expect(err).NotTo(HaveOccurred())

		router = gin.New()
		router.Use(bkerrs.ErrorHandler())
		image.Register(router.Group(""), New(&storereg.Registry{
			CustomRuntimeImageStore: store,
			SnapshotStore:           snapshotStore,
		}))

		mockey.Mock(ginperm.ValidateWorkspaceByID).To(
			func(
				_ context.Context, _ *storereg.Registry, wsID string, _ ginperm.Type,
			) (*workspace.Workspace, error) {
				return &workspace.Workspace{ID: wsID}, permErr
			},
		).Build()
		mockey.Mock(workspace.GetWorkspaceImageRegistry).Return(&bkmsreg.ImageRegistry{
			Registry: registryAddr, Username: "ws-user", Password: "ws-pass",
		}, nil).Build()
		mockey.Mock(taskq.Enqueue).Return(nil).Build()
		mockey.Mock(infrasreg.New).Return(&infrasreg.Client{}).Build()
		mockey.Mock((*infrasreg.Client).ListAllTags).To(
			func(*infrasreg.Client, context.Context, string) ([]string, error) {
				return remoteTags, remoteErr
			},
		).Build()
	})

	AfterEach(func() {
		Expect(store.DeleteAll(ctx)).To(Succeed())
		Expect(snapshotStore.DeleteAll(ctx)).To(Succeed())
		mockey.UnPatchAll()
	})

	// seedPersistedImage 造落库记录与新鲜快照，让请求走快照来源
	seedPersistedImage := func(tags ...string) {
		Expect(store.Upsert(ctx, &customruntime.Image{
			WorkspaceID: workspaceID, Type: customruntime.ImageTypeBuilder, Name: imageName,
		})).To(Succeed())
		Expect(snapshotStore.UpsertSnapshots(ctx, repoKey, lo.Map(tags,
			func(tag string, _ int) snapshot.Image { return snapshot.Image{Tag: tag} },
		))).To(Succeed())
		refreshedAt := time.Now()
		Expect(snapshotStore.UpsertStatus(ctx, &snapshot.RepoSnapshotStatus{
			RepoKey: repoKey, RepoName: imageName,
			RefreshStatus: snapshot.RefreshStatusIdle, LastRefreshedAt: &refreshedAt,
		})).To(Succeed())
	}

	serve := func(req *http.Request) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	listTags := func(name string) *httptest.ResponseRecorder {
		return serve(httptest.NewRequest(http.MethodGet, "/workspaces/"+workspaceID+
			"/custom-build-images/tags?page=1&pageSize=10&name="+url.QueryEscape(name), nil))
	}

	refreshTags := func(name string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(
			http.MethodPost,
			"/workspaces/"+workspaceID+"/custom-build-images/tags/refresh",
			strings.NewReader(`{"name":"`+name+`"}`),
		)
		req.Header.Set("Content-Type", "application/json")
		return serve(req)
	}

	tagsOf := func(rec *httptest.ResponseRecorder) *serializer.PaginatedCustomRuntimeImageTagOutputObjs {
		var resp serializer.ListCustomRuntimeImageTagsOutput
		Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(Succeed())
		return resp.Data
	}

	refreshResultOf := func(rec *httptest.ResponseRecorder) *serializer.RefreshResultInfoOutputObj {
		var resp serializer.RefreshCustomRuntimeImageTagsOutput
		Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(Succeed())
		return resp.Data
	}

	Describe("ListCustomBuildImageTags", func() {
		It("serves the realtime and snapshot sources through an identical response shape", func() {
			remoteTags = []string{"v1.0", "v2.0"}

			// 尚无落库记录，走实时远程拉取
			realtime := listTags(imageName)
			Expect(realtime.Code).To(Equal(http.StatusOK))

			// 落库后同一请求自动切到快照来源，出参必须同构
			seedPersistedImage("v1.0", "v2.0")
			persisted := listTags(imageName)
			Expect(persisted.Code).To(Equal(http.StatusOK))

			Expect(tagsOf(realtime).Count).To(Equal(tagsOf(persisted).Count))
			Expect(tagsOf(realtime).Results).To(HaveLen(len(tagsOf(persisted).Results)))
			// 实时来源没有快照记录，状态退化为 idle 而非缺失
			Expect(tagsOf(realtime).SnapshotStatus.RefreshStatus).
				To(Equal(string(snapshot.RefreshStatusIdle)))
		})

		It("maps failures to the expected HTTP status codes", func() {
			// 不属于本工作空间镜像源，归属校验先拦下，不会走到远程拉取
			Expect(listTags("evil.example.com/foo/bar").Code).To(Equal(http.StatusBadRequest))

			remoteErr = &transport.Error{StatusCode: http.StatusNotFound}
			Expect(listTags(imageName).Code).To(Equal(http.StatusNotFound))

			remoteErr = context.DeadlineExceeded
			rec := listTags(imageName)
			Expect(rec.Code).To(Equal(http.StatusInternalServerError))
			// bkerrs 无 deadline 语义的错误码，超时只能靠错误信息里的标识区分
			Expect(rec.Body.String()).To(ContainSubstring("timed out"))

			// 未落库镜像不能刷新，避免写出无人读的孤儿快照
			Expect(refreshTags(imageName).Code).To(Equal(http.StatusNotFound))

			permErr = bkerrs.New(bkerrs.ErrCodeIAMNoPermission, "no workspace permission")
			Expect(listTags(imageName).Code).To(Equal(http.StatusForbidden))
			Expect(refreshTags(imageName).Code).To(Equal(http.StatusForbidden))
		})
	})

	Describe("RefreshCustomBuildImageTags", func() {
		It("reports success, refreshing and registry failures all as 200", func() {
			remoteTags = []string{"v1.0", "v2.0"}
			seedPersistedImage("v1.0", "stale")

			rec := refreshTags(imageName)
			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(refreshResultOf(rec).Status).To(Equal(snapshot.RefreshResultSuccess))
			Expect(refreshResultOf(rec).AddedTagCnt).To(BeEquivalentTo(1))
			Expect(refreshResultOf(rec).RemovedTagCnt).To(BeEquivalentTo(1))

			// 已有刷新在进行中：返回 refreshing 而非报错
			Expect(snapshotStore.UpsertStatus(ctx, &snapshot.RepoSnapshotStatus{
				RepoKey: repoKey, RepoName: imageName, RefreshStatus: snapshot.RefreshStatusRefreshing,
			})).To(Succeed())
			rec = refreshTags(imageName)
			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(refreshResultOf(rec).Status).To(Equal(snapshot.RefreshResultRefreshing))

			// 仓库侧鉴权失败：折成 failed 结果，同样是 200；seedPersistedImage 顺带把状态复位
			remoteTags, remoteErr = nil, &transport.Error{StatusCode: http.StatusUnauthorized}
			seedPersistedImage("v1.0")
			rec = refreshTags(imageName)
			Expect(rec.Code).To(Equal(http.StatusOK))
			Expect(refreshResultOf(rec).Status).To(Equal(snapshot.RefreshResultFailed))
			Expect(refreshResultOf(rec).Message).NotTo(BeEmpty())
		})
	})
})
