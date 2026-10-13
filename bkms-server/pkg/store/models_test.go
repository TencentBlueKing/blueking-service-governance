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

package store_test

import (
	"context"
	"strings"
	"time"

	"github.com/TencentBlueKing/gopkg/stringx"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/store"
)

// 本文件定义测试用的数据模型，它们分别模拟了现有 Store 中的几种典型形态。

// Widget 模拟最常见的模型：ObjectID 主键、自然键唯一索引（ownerID + name）、
// 创建/更新时间戳、值类型嵌套结构体、指针嵌套的敏感字段。
type Widget struct {
	ID        bson.ObjectID `bson:"_id,omitempty"`
	OwnerID   string        `bson:"ownerID"`
	Name      string        `bson:"name"`
	Priority  int           `bson:"priority"`
	Tags      []string      `bson:"tags"`
	Spec      WidgetSpec    `bson:"spec"`
	Secret    *Credential   `bson:"secret,omitempty"`
	CreatedAt time.Time     `bson:"createdAt"`
	UpdatedAt time.Time     `bson:"updatedAt"`
}

// WidgetSpec 值类型嵌套结构体
type WidgetSpec struct {
	Color string `bson:"color"`
	Size  int    `bson:"size"`
}

// Credential 指针嵌套结构体，其中 Password 需要加密存储
type Credential struct {
	Username string `bson:"username"`
	Password string `bson:"password"`
}

const (
	widgetCollection = "store_test_widgets"
	encPrefix        = "enc:"
	defaultColor     = "gray"
)

var (
	ErrWidgetNotFound = errors.New("widget not found")
	ErrWidgetExists   = errors.New("widget already exists")
	errNameRequired   = errors.New("widget name is required")
	errNotEncrypted   = errors.New("password is not encrypted")
)

func requireWidgetName(w *Widget) error {
	if w.Name == "" {
		return errNameRequired
	}
	return nil
}

func defaultWidgetColor(w *Widget) error {
	if w.Spec.Color == "" {
		w.Spec.Color = defaultColor
	}
	return nil
}

// encryptWidgetPassword 模拟加密：加上固定前缀，便于直接断言库中实际存储的内容
func encryptWidgetPassword(w *Widget) error {
	if w.Secret != nil && w.Secret.Password != "" {
		w.Secret.Password = encPrefix + w.Secret.Password
	}
	return nil
}

func decryptWidgetPassword(w *Widget) error {
	if w.Secret == nil || w.Secret.Password == "" {
		return nil
	}
	plain, ok := strings.CutPrefix(w.Secret.Password, encPrefix)
	if !ok {
		return errNotEncrypted
	}
	w.Secret.Password = plain
	return nil
}

// widgetMeta 模拟带有校验、默认值与加解密逻辑的模型元信息
var widgetMeta = store.Meta[Widget]{
	Collection: widgetCollection,
	NotFound:   ErrWidgetNotFound,
	Duplicate:  ErrWidgetExists,
	BeforeSave: []func(*Widget) error{requireWidgetName, defaultWidgetColor},
	Encode:     []func(*Widget) error{encryptWidgetPassword},
	Decode:     []func(*Widget) error{decryptWidgetPassword},
}

// Item 模拟以字符串作为 _id、没有时间戳的模型（如 port-forward 白名单）
type Item struct {
	ID    string `bson:"_id"`
	Value string `bson:"value"`
}

const itemCollection = "store_test_items"

var itemMeta = store.Meta[Item]{Collection: itemCollection}

// Timestamps 以 inline 方式嵌入的公共时间戳
type Timestamps struct {
	CreatedAt time.Time `bson:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt"`
}

// Note 模拟通过 inline 嵌入时间戳的模型
type Note struct {
	ID         bson.ObjectID `bson:"_id,omitempty"`
	Timestamps `bson:",inline"`
	Body       string `bson:"body"`
}

const noteCollection = "store_test_notes"

var noteMeta = store.Meta[Note]{Collection: noteCollection}

// storeFixture 汇总测试用例需要的依赖
type storeFixture struct {
	ctx     context.Context
	app     *fxtest.App
	db      *mongo.Database
	widgets *store.GenericStore[Widget]
	items   *store.GenericStore[Item]
	notes   *store.GenericStore[Note]
	// owner 每个用例使用独立的 ownerID，避免用例之间的数据互相影响
	owner string
}

// setUpStores 通过 fx 提供测试模型的 Store，同时清理测试集合、确保唯一索引存在
func setUpStores() *storeFixture {
	f := &storeFixture{ctx: context.Background(), owner: "owner-" + stringx.Random(8)}
	f.app = fxtest.New(
		GinkgoT(),
		store.FxModule,
		store.Provide(widgetMeta),
		store.Provide(itemMeta),
		store.Provide(noteMeta),
		fx.Populate(&f.db, &f.widgets, &f.items, &f.notes),
	)
	f.app.RequireStart()

	for _, coll := range []*mongo.Collection{
		f.widgets.Collection(), f.items.Collection(), f.notes.Collection(),
	} {
		_, err := coll.DeleteMany(f.ctx, bson.M{})
		Expect(err).NotTo(HaveOccurred())
	}
	// 测试集合不在数据库迁移中，这里补上唯一索引，用于验证唯一键冲突的处理
	_, err := f.widgets.Collection().Indexes().CreateOne(f.ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "ownerID", Value: 1}, {Key: "name", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	Expect(err).NotTo(HaveOccurred())
	return f
}

func (f *storeFixture) tearDown() {
	f.app.RequireStop()
}

// newWidget 构造一个属于当前用例 owner 的 Widget
func (f *storeFixture) newWidget(name string) *Widget {
	return &Widget{OwnerID: f.owner, Name: name, Priority: 1, Tags: []string{"a"}}
}

// mustCreateWidget 创建并返回一个 Widget
func (f *storeFixture) mustCreateWidget(name string) *Widget {
	w := f.newWidget(name)
	Expect(f.widgets.Create(f.ctx, w)).To(Succeed())
	return w
}

// rawWidget 绕过 GenericStore 直接读取数据库中的原始文档，用于断言实际存储的内容。
// 返回 bson.Raw 而非 bson.M：驱动 v2 解码到 bson.M 时，嵌套文档默认是 bson.D，按路径取值更直接
func (f *storeFixture) rawWidget(name string) bson.Raw {
	raw, err := f.widgets.Collection().FindOne(f.ctx, bson.M{"ownerID": f.owner, "name": name}).Raw()
	Expect(err).NotTo(HaveOccurred())
	return raw
}

// rawString 读取原始文档中指定路径的字符串值
func rawString(raw bson.Raw, path ...string) string {
	v, err := raw.LookupErr(path...)
	Expect(err).NotTo(HaveOccurred())
	return v.StringValue()
}

// byOwner 匹配当前 owner 下的全部 Widget，模拟领域包中定义的具名 Opt
func (f *storeFixture) byOwner() store.Opt {
	return store.Where("ownerID", f.owner)
}

// byName 匹配当前 owner 下指定名称的 Widget，演示在已有 Opt 上组合新的 Opt
func (f *storeFixture) byName(name string) store.Opt {
	return store.Combine(f.byOwner(), store.Where("name", name))
}
