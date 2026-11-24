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

// Package ownership 为平台下发的 K8s 对象打归属标签，并在下发前校验集群中的同名对象是否属于本次下发方，
// 避免多个环境共用同一命名空间时互相覆盖资源。
package ownership

import (
	"context"
	"maps"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	envmodel "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/core/env/model"
	k8sclient "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/kubernetes/client"
)

const (
	// LabelKeyEnvName 标记对象所属环境名称
	LabelKeyEnvName = "io.tencent.bkms.env-name"
	// LabelKeyAppID 标记对象所属应用 ID
	LabelKeyAppID = "io.tencent.bkms.app-id"
)

// ErrNotOwned 集群中已有同名对象，且不属于本次下发的环境或应用
var ErrNotOwned = errors.New("object is not owned by the environment or application")

// Owner 对象的归属方
type Owner struct {
	EnvName string
	// AppID 为空表示环境级对象（如端口池），只校验环境
	AppID string
	// strict 为 true 时，集群中已有的无归属标签对象也视为他人所有。
	// 共用命名空间的环境开启：无标签对象只可能来自来源环境的历史部署或用户手动创建。
	strict bool
	// needVerify 为 false 时 Verify / Claim 不查询集群，只打标签。
	// 独占命名空间的环境里，同名对象只可能是本环境自己的，查询纯属额外开销。
	needVerify bool
}

// ForApp 返回应用在环境中下发对象的归属方
func ForApp(ctx context.Context, db *mongo.Database, env *envmodel.Environment, appID string) (Owner, error) {
	owner, err := ForEnv(ctx, db, env)
	if err != nil {
		return Owner{}, err
	}
	owner.AppID = appID
	return owner, nil
}

// ForEnv 返回环境级对象的归属方
func ForEnv(ctx context.Context, db *mongo.Database, env *envmodel.Environment) (Owner, error) {
	// 共用命名空间的环境：严格校验，无标签对象也视为他人所有
	if env.SharesNamespace() {
		return Owner{EnvName: env.Name, strict: true, needVerify: true}, nil
	}
	// 对于标准命名空间，只有其下有共用命名空间的环境时才做校验
	// 其他标准命名空间暂时认为不会有同资源冲突，避免在部署时引入太多额外查询
	sharers, err := envmodel.ListNamespaceSharers(ctx, db, env.Cluster.ClusterID, env.Cluster.Namespace)
	if err != nil {
		return Owner{}, errors.Wrapf(err, "resolve owner of env %s", env.Name)
	}
	return Owner{EnvName: env.Name, needVerify: len(sharers) > 0}, nil
}

// NeedVerify 返回下发前是否需要查询集群中的同名对象
func (o Owner) NeedVerify() bool {
	return o.needVerify
}

// Stamp 在对象 metadata 上打归属标签。不要传入 Pod 模板的 metadata，否则修改标签会触发滚动更新。
func (o Owner) Stamp(obj metav1.Object) {
	labels := maps.Clone(obj.GetLabels())
	if labels == nil {
		labels = make(map[string]string, 2)
	}
	labels[LabelKeyEnvName] = o.EnvName
	if o.AppID != "" {
		labels[LabelKeyAppID] = o.AppID
	}
	obj.SetLabels(labels)
}

// StampManifest 在 manifest 的 metadata 上打归属标签
func (o Owner) StampManifest(manifest map[string]any) {
	o.Stamp(&unstructured.Unstructured{Object: manifest})
}

// Check 校验集群中已有的同名对象 live 是否属于该归属方，live 不能为 nil
func (o Owner) Check(live metav1.Object) error {
	labels := live.GetLabels()
	envName, hasEnv := labels[LabelKeyEnvName]
	appID, hasApp := labels[LabelKeyAppID]
	if hasEnv && envName != o.EnvName {
		return errors.Wrapf(ErrNotOwned, "%s %q belongs to env %q", kindOf(live), live.GetName(), envName)
	}
	if o.AppID != "" && hasApp && appID != o.AppID {
		return errors.Wrapf(ErrNotOwned, "%s %q belongs to app %q", kindOf(live), live.GetName(), appID)
	}
	if o.strict && (!hasEnv || (o.AppID != "" && !hasApp)) {
		return errors.Wrapf(ErrNotOwned, "%s %q has no ownership labels", kindOf(live), live.GetName())
	}
	return nil
}

// Verify 读取集群中的同名对象并校验归属，避免影响同名但不同归属的对象
func (o Owner) Verify(ctx context.Context, cli *k8sclient.Client, namespace, name string) error {
	if !o.needVerify {
		return nil
	}
	live, err := cli.Get(ctx, namespace, name, metav1.GetOptions{})
	if errors.Is(err, k8sclient.ErrResourceNotFound) {
		return nil
	}
	if err != nil {
		return errors.Wrapf(err, "get object %s/%s for ownership check", namespace, name)
	}
	return o.Check(live)
}

// Claim 校验集群中同名对象的归属后，在 manifest 上打归属标签，避免有同名不同归属的对象
func (o Owner) Claim(ctx context.Context, cli *k8sclient.Client, namespace string, manifest map[string]any) error {
	obj := &unstructured.Unstructured{Object: manifest}
	if err := o.Verify(ctx, cli, namespace, obj.GetName()); err != nil {
		return err
	}
	o.Stamp(obj)
	return nil
}

func kindOf(obj metav1.Object) string {
	if u, ok := obj.(*unstructured.Unstructured); ok && u.GetKind() != "" {
		return u.GetKind()
	}
	return "object"
}
