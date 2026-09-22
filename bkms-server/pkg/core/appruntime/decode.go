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

package appruntime

import (
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/utils/mapstructurex"
)

// DecodeStrict decodes cfg into T, rejecting unknown keys and mismatched types at
// every level. The `json` struct tag names the keys, because a frameworkConfig map is
// both persisted as BSON and exposed as JSON under the same names.
//
// Decoding goes through mapstructure rather than a JSON round trip so that numbers
// keep their original kind: a field typed as `any` holds int32/int64 instead of the
// float64 that encoding/json would produce.
func DecodeStrict[T any](cfg map[string]any, version, expectedVersion int) (T, error) {
	var zero T

	if expectedVersion == 0 {
		return zero, errors.New("expected framework config version must be > 0")
	}
	if version != expectedVersion {
		if version == 0 {
			return zero, errors.Wrapf(ErrMissingConfigVersion, "expected %d", expectedVersion)
		}
		return zero, errors.Wrapf(ErrUnknownConfigVersion, "got %d, expected %d", version, expectedVersion)
	}

	var dest T
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:      &dest,
		TagName:     "json",
		ErrorUnused: true,
		DecodeHook:  mapstructurex.BsonDocToMapHook(),
	})
	if err != nil {
		return zero, errors.Wrap(err, "build frameworkConfig decoder")
	}
	if err = decoder.Decode(cfg); err != nil {
		return zero, errors.Wrap(err, "decode frameworkConfig")
	}
	return dest, nil
}

// FileMetaToMap copies fileName/filePath into a new map. Empty values are omitted.
func FileMetaToMap(fileName, filePath string) map[string]any {
	out := map[string]any{}
	if fileName != "" {
		out["fileName"] = fileName
	}
	if filePath != "" {
		out["filePath"] = filePath
	}
	return out
}

func fileMetaFromMap(cfg map[string]any) (fileName, filePath string) {
	if cfg == nil {
		return "", ""
	}
	fileName, _ = cfg["fileName"].(string)
	filePath, _ = cfg["filePath"].(string)
	return fileName, filePath
}
