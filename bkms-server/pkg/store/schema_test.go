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

package store

import (
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type schemaNamedID string

type SchemaBase struct {
	Owner string `bson:"owner"`
}

type schemaSpec struct {
	Color string `bson:"color"`
}

type (
	schemaDoc struct {
		ID         string `bson:"_id"`
		SchemaBase `bson:",inline"`
		Title      string     `bson:"title"`
		Spec       schemaSpec `bson:"spec"`
		CreatedAt  time.Time  `bson:"createdAt"`
	}
	schemaNoID struct {
		Name string `bson:"name"`
	}
	schemaNamedIDDoc struct {
		ID schemaNamedID `bson:"_id"`
	}
	schemaIntIDDoc struct {
		ID int64 `bson:"_id"`
	}
	schemaUpdatedOnly struct {
		UpdatedAt time.Time `bson:"updatedAt"`
	}
	schemaBadUpdatedAt struct {
		UpdatedAt string `bson:"updatedAt"`
	}
)

func mustSchema[T any]() *schema {
	return lo.Must(newSchema(reflect.TypeFor[T]()))
}

var _ = Describe("Schema", func() {
	Describe("newSchema", func() {
		It("should record keys of top-level and inline fields", func() {
			Expect(mustSchema[schemaDoc]().keys).To(Equal([]string{"_id", "owner", "title", "spec", "createdAt"}))
		})

		It("should reject a non-struct model", func() {
			_, err := newSchema(reflect.TypeFor[string]())
			Expect(err).To(MatchError(ContainSubstring("model must be a struct")))
		})

		It("should reject a timestamp field that is not time.Time", func() {
			_, err := newSchema(reflect.TypeFor[schemaBadUpdatedAt]())
			Expect(err).To(MatchError(ContainSubstring("must be time.Time, got string")))
		})
	})

	DescribeTable("resolveKey",
		func(field, expected string) {
			key, err := mustSchema[schemaDoc]().resolveKey(field)
			Expect(err).NotTo(HaveOccurred())
			Expect(key).To(Equal(expected))
		},
		Entry("converts Go names to bson keys", "Spec.Color", "spec.color"),
		Entry("keeps unknown fields as is", "legacy.Field", "legacy.Field"),
	)

	Describe("_id", func() {
		It("should generate an ObjectID hex for a named string type", func() {
			sc, doc := mustSchema[schemaNamedIDDoc](), &schemaNamedIDDoc{}
			sc.fillID(reflect.ValueOf(doc).Elem())
			Expect(bson.ObjectIDFromHex(string(doc.ID))).NotTo(BeZero())
		})

		It("should keep a preset ID", func() {
			sc, doc := mustSchema[schemaNamedIDDoc](), &schemaNamedIDDoc{ID: "preset"}
			sc.fillID(reflect.ValueOf(doc).Elem())
			Expect(doc.ID).To(Equal(schemaNamedID("preset")))
		})

		It("should leave an ID of an unsupported type to MongoDB", func() {
			sc, doc := mustSchema[schemaIntIDDoc](), &schemaIntIDDoc{}
			_, ok := sc.newID()
			Expect(ok).To(BeFalse())
			sc.fillID(reflect.ValueOf(doc).Elem())
			Expect(doc.ID).To(BeZero())
		})

		It("should convert the ID returned by the database to the field type", func() {
			sc, doc := mustSchema[schemaNamedIDDoc](), &schemaNamedIDDoc{}
			sc.setID(reflect.ValueOf(doc).Elem(), "from-db")
			Expect(doc.ID).To(Equal(schemaNamedID("from-db")))
		})

		It("should ignore an ID returned by the database that cannot be converted", func() {
			sc, doc := mustSchema[schemaNamedIDDoc](), &schemaNamedIDDoc{}
			sc.setID(reflect.ValueOf(doc).Elem(), bson.NewObjectID())
			Expect(doc.ID).To(BeZero())
		})

		It("should report a zero ID as absent", func() {
			sc := mustSchema[schemaNamedIDDoc]()
			_, ok := sc.idOf(reflect.ValueOf(schemaNamedIDDoc{}))
			Expect(ok).To(BeFalse())
			id, ok := sc.idOf(reflect.ValueOf(schemaNamedIDDoc{ID: "x"}))
			Expect(ok).To(BeTrue())
			Expect(id).To(Equal(schemaNamedID("x")))
		})

		It("should do nothing for models without _id", func() {
			sc, doc := mustSchema[schemaNoID](), &schemaNoID{Name: "n"}
			v := reflect.ValueOf(doc).Elem()
			sc.fillID(v)
			sc.setID(v, "from-db")
			_, hasNew := sc.newID()
			_, hasID := sc.idOf(v)
			Expect(hasNew).To(BeFalse())
			Expect(hasID).To(BeFalse())
			Expect(doc).To(Equal(&schemaNoID{Name: "n"}))
		})
	})

	Describe("timestamps", func() {
		now := time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC)

		It("should fill createdAt with now and keep updatedAt equal to it", func() {
			doc := &selDoc{}
			mustSchema[selDoc]().fillCreated(reflect.ValueOf(doc).Elem(), now)
			Expect(doc.CreatedAt).To(Equal(now))
			Expect(doc.UpdatedAt).To(Equal(now))
		})

		It("should keep a preset createdAt and use it as updatedAt", func() {
			preset := now.Add(-time.Hour)
			doc := &selDoc{CreatedAt: preset}
			mustSchema[selDoc]().fillCreated(reflect.ValueOf(doc).Elem(), now)
			Expect(doc.CreatedAt).To(Equal(preset))
			Expect(doc.UpdatedAt).To(Equal(preset))
		})

		It("should set updatedAt to now for models without createdAt", func() {
			doc := &schemaUpdatedOnly{}
			mustSchema[schemaUpdatedOnly]().fillCreated(reflect.ValueOf(doc).Elem(), now)
			Expect(doc.UpdatedAt).To(Equal(now))
		})

		It("should do nothing for models without timestamps", func() {
			doc := &schemaNoID{Name: "n"}
			sc := mustSchema[schemaNoID]()
			sc.fillCreated(reflect.ValueOf(doc).Elem(), now)
			sc.setUpdated(reflect.ValueOf(doc).Elem(), now)
			Expect(doc).To(Equal(&schemaNoID{Name: "n"}))
		})
	})
})
