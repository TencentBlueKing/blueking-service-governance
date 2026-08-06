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
)

// EditableContentField shows which field of the app config file can be edited by the user,
// for example, "Content" means the user may edit the "Content" field using the editor
// provided by the UI.
type EditableContentField string

const (
	// EditableContentFieldNone means no field is editable for current app config file.
	EditableContentFieldNone EditableContentField = "none"
	// EditableContentFieldContent means the "Content" field is editable.
	EditableContentFieldContent EditableContentField = "content"
	// EditableContentFieldOverlayContent means the "OverlayContent" field is editable.
	EditableContentFieldOverlayContent EditableContentField = "overlayContent"
)

// AppConfigFileEditor is an interface for editing the app config file.
//
// IMPORTANT: The Set* methods only modify the in-memory object, it's the caller's responsibility
// to persist the changes to the storage.
type AppConfigFileEditor interface {
	// GetEditableContentField returns the editable field of the app config file content.
	GetEditableContentField() EditableContentField

	// SetContent try to set the content of the app config file.
	SetContent(content string) error
	// SetOverlayContent try to set the overlay content of the app config file.
	SetOverlayContent(content string) error

	// GetCompiledContent returns the compiled content of the app config file.
	GetCompiledContent(ctx context.Context) (string, error)
}

// NewAppConfigFileEditor creates an editor object based on the given file object.
func NewAppConfigFileEditor(store AppConfigFileStore, acf *AppConfigFile) (AppConfigFileEditor, error) {
	switch acf.ContentSourceType {
	case ContentSourceTypeLocal:
		return newLocalAppConfigFileEditor(store, acf), nil
	case ContentSourceTypeBSCP:
		return newBSCPAppConfigFileEditor(store, acf), nil
	default:
		return nil, errors.New("invalid content source type")
	}
}

// LocalAppConfigFileEditor for ContentSourceType: local
type LocalAppConfigFileEditor struct {
	store AppConfigFileStore
	acf   *AppConfigFile
}

// newLocalAppConfigFileEditor creates a new LocalAppConfigFileEditor
func newLocalAppConfigFileEditor(store AppConfigFileStore, acf *AppConfigFile) AppConfigFileEditor {
	return &LocalAppConfigFileEditor{store: store, acf: acf}
}

// GetEditableContentField returns the editable field of the app config file content
func (e *LocalAppConfigFileEditor) GetEditableContentField() EditableContentField {
	switch e.acf.Type {
	case AppConfigFileTypeNormal:
		return EditableContentFieldContent
	case AppConfigFileTypeOverlay:
		return EditableContentFieldOverlayContent
	default:
		return EditableContentFieldNone
	}
}

// SetContent ...
func (e *LocalAppConfigFileEditor) SetContent(content string) error {
	if e.acf.Type != AppConfigFileTypeNormal {
		return errors.New("only normal app config file can set")
	}
	e.acf.Content = &content
	return nil
}

// SetOverlayContent ...
func (e *LocalAppConfigFileEditor) SetOverlayContent(content string) error {
	if e.acf.Type != AppConfigFileTypeOverlay {
		return errors.New("only overlay app config file can set")
	}
	e.acf.OverlayContent = &content
	return nil
}

// GetCompiledContent ...
func (e *LocalAppConfigFileEditor) GetCompiledContent(ctx context.Context) (string, error) {
	switch e.acf.Type {
	// Normal 类型，直接取 Content 字段内容
	case AppConfigFileTypeNormal:
		if e.acf.Content == nil {
			return "", errors.New("app config file content is empty")
		}
		return *e.acf.Content, nil
	// Overlay 类型，需要先获取 BaseContent，再与 OverlayContent 合并
	case AppConfigFileTypeOverlay:
		provider, err := NewBaseContentProvider(e.store, e.acf)
		if err != nil {
			return "", errors.Wrap(err, "create base content provider")
		}

		info, err := provider.GetInfo(ctx)
		if err != nil {
			return "", errors.Wrap(err, "get base content info")
		}

		result, err := mergeContent(info.Content, e.acf.OverlayContent, e.acf.GetConfigFormat())
		if err != nil {
			return "", errors.Wrap(err, "merge content")
		}
		return result, nil
	}

	return "", errors.Errorf("invalid app config file type: %s", e.acf.Type)
}

var _ AppConfigFileEditor = (*LocalAppConfigFileEditor)(nil)

// BSCPAppConfigFileEditor for ContentSourceType: BSCP
type BSCPAppConfigFileEditor struct {
	store AppConfigFileStore
	acf   *AppConfigFile
}

// newBSCPAppConfigFileEditor creates a new BSCPAppConfigFileEditor.
func newBSCPAppConfigFileEditor(store AppConfigFileStore, acf *AppConfigFile) AppConfigFileEditor {
	return &BSCPAppConfigFileEditor{store: store, acf: acf}
}

// GetEditableContentField returns the editable field of the app config file content
func (e *BSCPAppConfigFileEditor) GetEditableContentField() EditableContentField {
	switch e.acf.Type {
	case AppConfigFileTypeNormal:
		// For BSCP values, only editing the OverlayContent of normal file is supported.
		return EditableContentFieldOverlayContent
	default:
		return EditableContentFieldNone
	}
}

// SetContent set the content field for BSCP app config file, always return error
// because writing content is not allowed for bscp source type.
func (e *BSCPAppConfigFileEditor) SetContent(_ string) error {
	return errors.New("unable to set content for bscp source type")
}

// SetOverlayContent set the overlay content field for BSCP app config file, only allowed for normal
// app config file. The overlay content is used for patching the original content.
func (e *BSCPAppConfigFileEditor) SetOverlayContent(content string) error {
	if e.acf.Type == AppConfigFileTypeOverlay {
		return errors.New("only normal app config file can set overlay content for bscp source type")
	}
	e.acf.OverlayContent = &content
	return nil
}

// GetCompiledContent returns the compiled content for BSCP app config file.
func (e *BSCPAppConfigFileEditor) GetCompiledContent(ctx context.Context) (string, error) {
	switch e.acf.Type {
	// Normal 类型，直接从 BSCP 取配置内容，再与 AppConfigFile 中的 OverlayContent 合并
	case AppConfigFileTypeNormal:
		content, err := e.acf.BSCPConfig.FetchContent(ctx)
		if err != nil {
			return "", errors.Wrap(err, "fetch content from BSCP")
		}

		result, err := mergeContent(content, e.acf.OverlayContent, e.acf.GetConfigFormat())
		if err != nil {
			return "", errors.Wrap(err, "merge content")
		}
		return result, nil

	// Overlay 类型，需要先获取 BaseContent，再从 bscp 取配置内容合并
	case AppConfigFileTypeOverlay:
		provider, err := NewBaseContentProvider(e.store, e.acf)
		if err != nil {
			return "", errors.Wrap(err, "create base content provider")
		}

		info, err := provider.GetInfo(ctx)
		if err != nil {
			return "", errors.Wrap(err, "get base content info")
		}

		overlayContent, err := e.acf.BSCPConfig.FetchContent(ctx)
		if err != nil {
			return "", errors.Wrap(err, "fetch content from BSCP")
		}

		result, err := mergeContent(info.Content, &overlayContent, e.acf.GetConfigFormat())
		if err != nil {
			return "", errors.Wrap(err, "merge content")
		}

		return result, nil
	}

	return "", errors.Errorf("invalid app config file type: %s", e.acf.Type)
}

var _ AppConfigFileEditor = (*BSCPAppConfigFileEditor)(nil)
