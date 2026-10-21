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

package appcfg

import (
	"bytes"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/openapi"
	"sigs.k8s.io/kustomize/kyaml/yaml"
	"sigs.k8s.io/kustomize/kyaml/yaml/merge2"
	"sigs.k8s.io/kustomize/kyaml/yaml/walk"
)

// mergeContent merges two config contents based on the specified format.
// This is a convenience function that selects the appropriate merge function.
func mergeContent(base string, overlay *string, format FileFormat) (string, error) {
	switch format {
	case FileFormatTAF:
		return mergeTAFContent(base, overlay)
	default:
		// Default to YAML format
		return mergeYAMLContent(base, overlay)
	}
}

// mergeYAMLContent merges two YAML contents, with overlay taking precedence over base.
// This function shares the implementation with kustomize's strategic merge patch.
//
// The function supports two overlay formats:
//  1. Version 2 format (when overlayVersion: "2" is present):
//     ```
//     overlayVersion: "2"
//     patches:
//     - replicas: 5
//     - image: myapp:v2.0
//     ```
//  2. Simple format (default or version "1", for better user experience in simple cases):
//     ```
//     replicas: 5
//     image: myapp:v2.0
//     ```
//
// Unlike the vanilla strategic merge patch, null fields are preserved on both sides: a null
// field in base stays null, and a null field in overlay sets the field to null instead of
// deleting it. Use the `$patch: delete` directive to delete a map field. An empty base
// document (empty, blank, comment-only, `---` or `null`) is treated as an empty map.
func mergeYAMLContent(base string, overlay *string) (string, error) {
	if lo.FromPtr(overlay) == "" {
		return base, nil
	}
	var overlayMap map[string]any
	if err := yaml.Unmarshal([]byte(*overlay), &overlayMap); err != nil {
		return "", errors.Wrap(err, "parsing the overlay YAML")
	}

	// Read the overlayVersion
	overlayVersion, hasVersion := overlayMap["overlayVersion"].(string)
	if !hasVersion {
		overlayVersion = "1"
	}

	var patches []any
	switch overlayVersion {
	case "1":
		// Simple format: treat the whole overlay as a single patch
		patches = []any{overlayMap}
	case "2":
		// Version 2 format: use patches field
		var ok bool
		patches, ok = overlayMap["patches"].([]any)
		if patches == nil || !ok {
			// The "patches" field is missing or in the wrong type
			return base, nil
		}
	default:
		return "", errors.Errorf("unsupported overlayVersion: %s, supported: '1'(default), '2'", overlayVersion)
	}

	// Apply each patch in sequence
	currentResult := base
	for _, patch := range patches {
		// No need to handle the error since the patch is from parsing step above
		patchStr, _ := yaml.Marshal(patch)

		// Parse the overlay YAML
		overlayNode, err := yaml.Parser{Value: string(patchStr)}.Filter(nil)
		if err != nil {
			return "", errors.Wrap(err, "parsing the overlay YAML")
		}

		currentResult, err = applyYAMLPatch(currentResult, overlayNode)
		if err != nil {
			return "", errors.Wrap(err, "merging the YAML contents")
		}
	}
	return currentResult, nil
}

// applyYAMLPatch applies a strategic merge patch to every document in base.
func applyYAMLPatch(base string, patch *yaml.RNode) (string, error) {
	docs, err := (&kio.ByteReader{
		Reader:                bytes.NewBufferString(base),
		OmitReaderAnnotations: true,
	}).Read()
	if err != nil {
		return "", errors.Wrap(err, "parsing the base YAML")
	}
	if len(docs) == 0 {
		docs = []*yaml.RNode{yaml.NewMapRNode(nil)}
	}

	var merged []*yaml.RNode
	for _, doc := range docs {
		result, err := walk.Walker{
			Sources: []*yaml.RNode{doc, patch},
			Visitor: nullPreservingMerger{},
			MergeOptions: yaml.MergeOptions{
				ListIncreaseDirection: yaml.MergeOptionsListPrepend,
			},
		}.Walk()
		if err != nil {
			return "", errors.Wrap(err, "applying the patch")
		}
		if result != nil {
			merged = append(merged, result)
		}
	}

	var buf bytes.Buffer
	if err := (kio.ByteWriter{Writer: &buf}).Write(merged); err != nil {
		return "", errors.Wrap(err, "writing the merged YAML")
	}
	return buf.String(), nil
}

// nullPreservingMerger is a strategic merge visitor which keeps null fields rather than clearing them.
type nullPreservingMerger struct {
	merge2.Merger
}

var _ walk.Visitor = nullPreservingMerger{}

func (m nullPreservingMerger) VisitMap(nodes walk.Sources, s *openapi.ResourceSchema) (*yaml.RNode, error) {
	if n := keptNullNode(nodes); n != nil {
		return n, nil
	}
	return m.Merger.VisitMap(nodes, s)
}

func (m nullPreservingMerger) VisitScalar(nodes walk.Sources, s *openapi.ResourceSchema) (*yaml.RNode, error) {
	if n := keptNullNode(nodes); n != nil {
		return n, nil
	}
	return m.Merger.VisitScalar(nodes, s)
}

func (m nullPreservingMerger) VisitList(
	nodes walk.Sources, s *openapi.ResourceSchema, kind walk.ListKind,
) (*yaml.RNode, error) {
	if n := keptNullNode(nodes); n != nil {
		return n, nil
	}
	return m.Merger.VisitList(nodes, s, kind)
}

// keptNullNode returns a null node that survives the merge when the patch sets the field to
// null, or when the patch leaves the field untouched and it is null in base. Otherwise, it
// returns nil.
func keptNullNode(nodes walk.Sources) *yaml.RNode {
	var kept *yaml.RNode
	switch origin, dest := nodes.Origin(), nodes.Dest(); {
	case origin.IsTaggedNull():
		kept = yaml.NewRNode(origin.YNode())
	case origin.IsNil() && dest.IsTaggedNull():
		kept = yaml.NewRNode(dest.YNode())
	default:
		return nil
	}
	kept.ShouldKeep = true
	return kept
}
