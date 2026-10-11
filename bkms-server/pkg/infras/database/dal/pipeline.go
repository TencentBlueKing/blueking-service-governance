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

package dal

import (
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/tenant"
)

// allowedStages 租户表聚合只允许单表阶段。跨表或写出结果的阶段会被拒绝。
var allowedStages = map[string]struct{}{
	"$match":   {},
	"$group":   {},
	"$sort":    {},
	"$project": {},
	"$skip":    {},
	"$limit":   {},
	"$count":   {},
	"$unwind":  {},
}

func applyPipeline(pipeline any, tenantID string) (bson.A, error) {
	stages, err := pipelineStages(pipeline)
	if err != nil {
		return nil, err
	}
	for _, stage := range stages {
		if len(stage) == 0 {
			return nil, errors.New("pipeline stage is empty")
		}
		name := stage[0].Key
		if _, ok := allowedStages[name]; !ok {
			return nil, errors.Errorf("pipeline stage %s is not allowed on tenant table", name)
		}
	}

	match, err := setTenantField(bson.M{}, tenantID)
	if err != nil {
		return nil, err
	}
	out := make(bson.A, 0, len(stages)+1)
	out = append(out, bson.D{{Key: "$match", Value: match}})
	for _, stage := range stages {
		out = append(out, stage)
	}
	return out, nil
}

func pipelineStages(pipeline any) ([]bson.D, error) {
	switch value := pipeline.(type) {
	case mongo.Pipeline:
		return []bson.D(value), nil
	case []bson.D:
		return value, nil
	case bson.A:
		stages := make([]bson.D, 0, len(value))
		for _, item := range value {
			stage, ok := item.(bson.D)
			if !ok {
				return nil, errors.Errorf("pipeline stage %T is not bson.D", item)
			}
			stages = append(stages, stage)
		}
		return stages, nil
	default:
		return nil, errors.Errorf("pipeline %T is not supported", pipeline)
	}
}

func ensureTenantImmutable(update any) error {
	switch value := update.(type) {
	case bson.A, []bson.D, mongo.Pipeline:
		return errors.New("pipeline-style update is not allowed on tenant table")
	case bson.M:
		return rejectTenantOperators(value)
	case bson.D:
		return rejectTenantOperators(value)
	default:
		return errors.Errorf("update %T is not supported", update)
	}
}

func rejectTenantOperators(update any) error {
	ops, err := updateOperators(update)
	if err != nil {
		return err
	}
	for _, op := range []string{"$set", "$unset", "$rename"} {
		fields, ok := ops[op]
		if !ok {
			continue
		}
		names, err := fieldNames(fields)
		if err != nil {
			return err
		}
		for _, name := range names {
			if name == tenant.FieldTenantID {
				return errors.Errorf("update must not modify %s", tenant.FieldTenantID)
			}
		}
	}
	return nil
}

func updateOperators(update any) (map[string]any, error) {
	switch value := update.(type) {
	case bson.M:
		out := make(map[string]any, len(value))
		for key, item := range value {
			out[key] = item
		}
		return out, nil
	case bson.D:
		out := make(map[string]any, len(value))
		for _, elem := range value {
			out[elem.Key] = elem.Value
		}
		return out, nil
	default:
		return nil, errors.Errorf("update %T is not supported", update)
	}
}

func fieldNames(value any) ([]string, error) {
	switch fields := value.(type) {
	case bson.M:
		names := make([]string, 0, len(fields))
		for key := range fields {
			names = append(names, key)
		}
		return names, nil
	case map[string]any:
		names := make([]string, 0, len(fields))
		for key := range fields {
			names = append(names, key)
		}
		return names, nil
	case bson.D:
		names := make([]string, 0, len(fields))
		for _, elem := range fields {
			names = append(names, elem.Key)
		}
		return names, nil
	default:
		return nil, errors.Errorf("operator value %T is not supported", value)
	}
}
