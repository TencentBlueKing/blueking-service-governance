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

// Package initdata is responsible for seeding the database with a predefined set of
// service definitions.
//
// It reads service configurations from embedded JSON files ("service-general.json"),
// validates them against corresponding JSON schemas for structural integrity,
// and then parses them into the service models.
package initdata

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/xeipuuv/gojsonschema"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/extension/depservice/model"
)

// Do initialize service data into database
func Do(store model.ServiceStore) error {
	services, err := parseServices()
	if err != nil {
		return errors.Wrap(err, "parse services")
	}

	ctx := context.Background()

	for _, svc := range services {
		if err = store.Upsert(ctx, svc); err != nil {
			return errors.Wrapf(err, "upsert service %s", svc.Name)
		}
	}

	return nil
}

// parseServices parses services from service-general.json
func parseServices() (map[string]*model.Service, error) {
	svcJson, err := validateService("service-general.json", "service-general-schema.json")
	if err != nil {
		return nil, err
	}

	var initData []map[string]any

	if err = json.Unmarshal(svcJson, &initData); err != nil {
		return nil, errors.Wrap(err, "unmarshal init data")
	}

	svcMap := make(map[string]*model.Service)
	for _, d := range initData {
		plans := make([]model.ServicePlan, 0)
		// existingPlans 用于检测同一服务下，是否有重复的 plan name
		existingPlans := make(map[string]struct{})

		for _, p := range d["plans"].([]any) {
			plan := new(model.ServicePlan)
			if err = mapstructure.Decode(p, plan); err != nil {
				return nil, errors.Wrap(err, "decode plan")
			}
			if _, exists := existingPlans[plan.Name]; exists {
				return nil, errors.Errorf(
					"duplicate plan name '%s' in service '%s'",
					plan.Name, d["name"],
				)
			}
			existingPlans[plan.Name] = struct{}{}
			plans = append(plans, *plan)
		}

		svcName := strings.ToLower(d["name"].(string))
		svcMap[svcName] = &model.Service{
			Name:        strings.ToLower(d["name"].(string)),
			Category:    d["category"].(string),
			DisplayName: d["displayName"].(string),
			Description: d["description"].(string),
			Plans:       plans,
		}
	}
	return svcMap, nil
}

// validateService validates the service json file and returns the json data
func validateService(jsonFile, schemaFile string) ([]byte, error) {
	rawJson, err := AuthScopesFS.ReadFile(jsonFile)
	if err != nil {
		return nil, errors.Wrapf(err, "read %s", jsonFile)
	}

	schemaJson, err := AuthScopesFS.ReadFile(schemaFile)
	if err != nil {
		return nil, errors.Wrapf(err, "read %s", schemaFile)
	}

	jsonLoader := gojsonschema.NewStringLoader(string(rawJson))
	schemaLoader := gojsonschema.NewStringLoader(string(schemaJson))

	result, err := gojsonschema.Validate(schemaLoader, jsonLoader)
	if err != nil {
		return nil, errors.Errorf("schema validation error: %v", err)
	}

	if !result.Valid() {
		errMsg := make([]string, 0)
		for _, desc := range result.Errors() {
			errMsg = append(errMsg, desc.String())
		}
		return nil, errors.New(strings.Join(errMsg, "\n"))
	}
	return rawJson, nil
}
