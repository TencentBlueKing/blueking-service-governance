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

package handler_test

import (
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	bcsmanager "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bcs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/cmdb"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/workspace/handler"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkcc"
)

var _ = Describe("ClassifyEnsureBkSystemsErr", func() {
	It("maps missing independent biz id to invalid request", func() {
		code := handler.ClassifyEnsureBkSystemsErr(workspace.ErrIndependentWorkspaceBizRequired)
		Expect(code).To(Equal(bkerrs.ErrCodeInvalidRequest))
	})

	It("maps inaccessible biz to invalid request", func() {
		err := fmt.Errorf("%w: bkcc biz 398", workspace.ErrIndependentWorkspaceBizInvalid)
		code := handler.ClassifyEnsureBkSystemsErr(err)
		Expect(code).To(Equal(bkerrs.ErrCodeInvalidRequest))
	})

	It("maps invalid bound bcs project to invalid request", func() {
		err := fmt.Errorf("%w: bcs project(%s) has no associated bizID", workspace.ErrBoundBCSProjectInvalid, "demo")
		code := handler.ClassifyEnsureBkSystemsErr(err)
		Expect(code).To(Equal(bkerrs.ErrCodeInvalidRequest))
	})

	It("maps existing deterministic bcs project to already exists", func() {
		err := fmt.Errorf("%w: bcs project(%s) already exists", workspace.ErrBCSProjectAlreadyExists, "bkms-ws-1")
		code := handler.ClassifyEnsureBkSystemsErr(err)
		Expect(code).To(Equal(bkerrs.ErrCodeAlreadyExists))
	})

	It("maps bound bcs project to already exists", func() {
		err := fmt.Errorf(
			"%w: bcs project(%s) already bound by workspace %s",
			workspace.ErrBCSProjectAlreadyBound,
			"existing-bcs",
			"other-ws",
		)
		code := handler.ClassifyEnsureBkSystemsErr(err)
		Expect(code).To(Equal(bkerrs.ErrCodeAlreadyExists))
	})

	DescribeTable("maps biz problems in shared project mode to invalid request",
		func(sentinel error) {
			err := fmt.Errorf("%w: get cmdb info of biz 398", sentinel)
			Expect(handler.ClassifyEnsureBkSystemsErr(err)).To(Equal(bkerrs.ErrCodeInvalidRequest))
		},
		Entry("biz not found or not accessible", bkcc.ErrBusinessNotFound),
		Entry("biz has no level-2 business", cmdb.ErrBusinessLevel2Missing),
	)

	DescribeTable("maps occupied projects to already exists",
		func(sentinel error) {
			err := fmt.Errorf("%w: project already occupied", sentinel)
			Expect(handler.ClassifyEnsureBkSystemsErr(err)).To(Equal(bkerrs.ErrCodeAlreadyExists))
		},
		Entry("bkci project code used by another workspace", bkci.ErrProjectCodeAlreadyUsed),
		Entry("workspace already bound to another bkci project", bkci.ErrWorkspaceAlreadyHasProject),
		Entry("bcs project with the same code belongs to another biz", bcsmanager.ErrProjectBizMismatch),
	)

	DescribeTable("maps workspace insert conflicts to already exists",
		func(sentinel error) {
			err := fmt.Errorf("%w: create workspace demo", sentinel)
			Expect(handler.ClassifyCreateWorkspaceErr(err)).To(Equal(bkerrs.ErrCodeAlreadyExists))
		},
		Entry("workspace id already used", workspace.ErrWorkspaceAlreadyExists),
		Entry("bcs project already bound", workspace.ErrBCSProjectAlreadyBound),
	)

	It("keeps unexpected insert errors as internal server error", func() {
		code := handler.ClassifyCreateWorkspaceErr(errors.New("create workspace demo"))
		Expect(code).To(Equal(bkerrs.ErrCodeInternalServerError))
	})

	It("keeps unexpected errors as internal server error", func() {
		code := handler.ClassifyEnsureBkSystemsErr(errors.New("init bkrepo project"))
		Expect(code).To(Equal(bkerrs.ErrCodeInternalServerError))
	})
})
