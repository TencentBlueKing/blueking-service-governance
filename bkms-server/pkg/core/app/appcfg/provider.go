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
	"context"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// BaseContentProvider defines the interface for providing base content information
// for app config files. This interface is responsible for retrieving the base content
// that serves as the foundation for overlay operations or content compilation.
type BaseContentProvider interface {
	// GetInfo returns the base content information for an app config file.
	// For overlay type app config files, this retrieves content from the base app config file.
	// For normal BSCP app config files, this can use the remote BSCP config as base content.
	//
	// Returns ErrBaseContentEmpty if no base content is available.
	GetInfo(ctx context.Context) (*BaseContentInfo, error)
}

// NewBaseContentProvider creates provider object based on the given file object.
func NewBaseContentProvider(store AppConfigFileStore, acf *AppConfigFile) (BaseContentProvider, error) {
	switch acf.ContentSourceType {
	case ContentSourceTypeLocal:
		return newLocalBaseContentProvider(store, acf), nil
	case ContentSourceTypeBSCP:
		return newBSCPBaseContentProvider(store, acf), nil
	default:
		return nil, errors.New("invalid content source type")
	}
}

// BaseContentInfo represents the information of the base content, some app config files such as
// local overlay type need the base content for compiling/rendering the final result.
type BaseContentInfo struct {
	// HolderID is the ID of the holder, "holder" is the app config file that holds current base content,
	// an app config file can function as its own base content holder or reference another file.
	HolderID bson.ObjectID
	// HolderName is the name of the holder
	HolderName string
	// HolderContentSourceType is the content source type of the holder, "local" or "bscp"
	HolderContentSourceType string

	// Content is the (compiled) base content of the holder
	Content string

	// IsFromAnotherFile indicates whether the base content is from another app config file,
	// for local overlay file, this field is always true.
	IsFromAnotherFile bool
}

// ErrBaseContentEmpty indicates the base content is empty.
var ErrBaseContentEmpty = errors.New("base content is empty")

// LocalBaseContentProvider provides base content for local content source app config files
type LocalBaseContentProvider struct {
	store AppConfigFileStore
	acf   *AppConfigFile
}

// newLocalBaseContentProvider creates a new LocalBaseContentProvider
func newLocalBaseContentProvider(store AppConfigFileStore, acf *AppConfigFile) BaseContentProvider {
	return &LocalBaseContentProvider{store: store, acf: acf}
}

// GetInfo returns the base content info for local app config files
func (p *LocalBaseContentProvider) GetInfo(ctx context.Context) (*BaseContentInfo, error) {
	if p.acf.Type != AppConfigFileTypeOverlay {
		// Non-overlay app config file has no base content
		return nil, ErrBaseContentEmpty
	}

	baseAcf, err := p.store.GetByID(ctx, *p.acf.BaseAppConfigFileID)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving the base app config file")
	}
	// 获取 Base AppConfigFile 的 Content
	editor, err := NewAppConfigFileEditor(p.store, baseAcf)
	if err != nil {
		return nil, errors.Wrap(err, "creating the base content provider")
	}
	compiledContent, err := editor.GetCompiledContent(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "compiling the base app config file content")
	}
	return &BaseContentInfo{
		HolderID:                baseAcf.ID,
		HolderName:              baseAcf.Name,
		HolderContentSourceType: string(baseAcf.ContentSourceType),
		IsFromAnotherFile:       true,
		Content:                 compiledContent,
	}, nil
}

var _ BaseContentProvider = &LocalBaseContentProvider{}

// BSCPBaseContentProvider provides base content for BSCP content source app config files
type BSCPBaseContentProvider struct {
	store AppConfigFileStore
	acf   *AppConfigFile
}

// newBSCPBaseContentProvider creates a new BSCPBaseContentProvider
func newBSCPBaseContentProvider(store AppConfigFileStore, acf *AppConfigFile) BaseContentProvider {
	return &BSCPBaseContentProvider{store: store, acf: acf}
}

// GetInfo 返回 AppConfigFile 的 BaseContentInfo
// 对于 overlay 类型的 AppConfigFile，其 BaseContent 来自另一个 AppConfigFile
// 对于 normal 类型的 BSCP AppConfigFile，支持基于远程 bscp 上的配置作为 BaseContent
// 具体设计可以参考 design_notes/multiple_values.md
func (p *BSCPBaseContentProvider) GetInfo(ctx context.Context) (*BaseContentInfo, error) {
	var targetAcf *AppConfigFile
	var isFromAnotherFile bool
	var err error

	if p.acf.Type == AppConfigFileTypeOverlay {
		// For overlay app config file, the base content is from another app config file
		targetAcf, err = p.store.GetByID(ctx, *p.acf.BaseAppConfigFileID)
		if err != nil {
			return nil, errors.Wrap(err, "retrieving the base app config file")
		}
		isFromAnotherFile = true
	} else {
		// For normal app config file, use itself as the base content holder
		targetAcf = p.acf
		isFromAnotherFile = false
	}

	// 获取 Base AppConfigFile 的 Content
	editor, err := NewAppConfigFileEditor(p.store, targetAcf)
	if err != nil {
		return nil, errors.Wrap(err, "creating the base content provider")
	}
	compiledContent, err := editor.GetCompiledContent(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "compiling the app config file content")
	}

	return &BaseContentInfo{
		HolderID:                targetAcf.ID,
		HolderName:              targetAcf.Name,
		HolderContentSourceType: string(targetAcf.ContentSourceType),
		IsFromAnotherFile:       isFromAnotherFile,
		Content:                 compiledContent,
	}, nil
}

var _ BaseContentProvider = &BSCPBaseContentProvider{}
