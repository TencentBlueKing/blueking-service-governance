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

package bkci

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	log "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/logging"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	cloudbkci "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/database"
)

const (
	// buildTriggerCallbackPathTmpl 触发回调相对路径（拼在 httpServer.publicBaseURL 之后；属 bkms API，非流水线模板字段）
	buildTriggerCallbackPathTmpl = "/bkms/v1/bkms-server/apps/%s/build-trigger-policies/callback"
	// buildTriggerCredentialIDPrefix 回调凭证 ID 前缀
	buildTriggerCredentialIDPrefix = "bkms_bt_" // #nosec G101
	// buildTriggerCredentialIDMaxLen 蓝盾凭证 ID 长度上限（仅字母、数字、下划线）
	buildTriggerCredentialIDMaxLen = 40
	// buildTriggerTokenBytes 回调 token 随机字节数
	buildTriggerTokenBytes = 32
	// buildTriggerRenderName 触发流水线显示名模板，仅用于 text/template 错误信息
	buildTriggerRenderName = "build-trigger:name"
	// buildTriggerRenderStages 触发流水线 stages 模板，仅用于 text/template 错误信息
	buildTriggerRenderStages = "build-trigger:stages"
)

// nonCredentialIDCharPattern 蓝盾凭证 ID 非法字符（含连字符）；生成 credentialID 时替换为下划线
var nonCredentialIDCharPattern = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

// TriggerPipelineManager 按应用管理触发专用流水线（Ensure / Cleanup）
//
// 与 PipelineManager 分叉：共享内置流水线走 Initialize；触发专用必须经本 Manager，
// 以便注入回调地址、应用独享凭证，并渲染带 appID 的显示名
type TriggerPipelineManager struct {
	workspaceID string
}

// NewTriggerPipelineManager 按工作空间创建触发流水线管理器
func NewTriggerPipelineManager(workspaceID string) *TriggerPipelineManager {
	return &TriggerPipelineManager{workspaceID: workspaceID}
}

// Ensure 确保指定应用的触发专用流水线存在（幂等 create-if-missing）
//
// 策略管理侧应在应用首条触发策略创建前同步调用；失败时调用方不得落库策略
//
// 本地已存在时直接返回，不做模板 semver 升级：共享流水线的
// ensureBuiltinPipelineTemplateVersion 会用模板 name/stages 全量覆盖蓝盾侧，
// 会冲掉本方法注入的显示名与回调脚本，以及触发器同步子需求写入的 Git 条件
// （不是循环依赖，而是多写入方共存下不能整模板覆盖；若需滚动模板应另做合并式 Sync）
// 与 PipelineManager.Initialize 对 build-trigger-* 的分叉理由一致，详见该函数注释
func (m *TriggerPipelineManager) Ensure(ctx context.Context, appID string) (*Pipeline, error) {
	// 本地以 workspaceID + build-trigger-{appID} 唯一索引幂等判定
	pipelineType := string(BuildTriggerPipelineType(appID))
	store, err := NewPipelineStoreMongo(database.Client(), database.Name())
	if err != nil {
		return nil, errors.Wrap(err, "create pipeline store")
	}
	existing, err := store.GetByWorkspaceAndType(ctx, m.workspaceID, pipelineType)
	if err == nil {
		// 幂等命中：有意跳过模板版本检查，理由见方法注释
		return existing, nil
	}
	if !errors.Is(err, ErrPipelineNotFound) {
		return nil, errors.Wrapf(err, "get workspace %s pipeline %s", m.workspaceID, pipelineType)
	}

	callbackURL, err := m.buildCallbackURL(appID)
	if err != nil {
		return nil, err
	}

	// 触发流水线挂在工作空间已绑定的蓝盾项目下
	projectStore, err := NewProjectStoreMongo(database.Client(), database.Name())
	if err != nil {
		return nil, errors.Wrap(err, "create project store")
	}
	project, err := projectStore.GetByWorkspace(ctx, m.workspaceID)
	if err != nil {
		return nil, errors.Wrapf(err, "get project by workspace %s", m.workspaceID)
	}

	client, err := cloudbkci.New(auth.MustGetUser(ctx))
	if err != nil {
		return nil, errors.Wrap(err, "create bkci client")
	}

	// 凭证 ID 对 appID 稳定可推导，便于 Cleanup / 失败重试时回收同名资源
	credentialID := m.buildCredentialID(appID)
	token, err := m.generateCallbackToken()
	if err != nil {
		return nil, errors.Wrap(err, "generate build-trigger callback token")
	}

	// 本地无流水线记录时，先清理可能残留的同名凭证，避免上次创建失败留下的孤儿凭证导致冲突
	if delErr := client.DeleteCredential(ctx, project.Code, credentialID); delErr != nil &&
		!errors.Is(delErr, cloudbkci.ObjectNotFound) {
		log.Warnf(
			ctx, "pre-clean stale build-trigger credential %s in project %s: %v", credentialID, project.Code, delErr,
		)
	}

	credDesc := fmt.Sprintf("bkms build-trigger callback credential for app %s", appID)
	if err = client.CreateAccessTokenCredential(ctx, project.Code, credentialID, credDesc, token); err != nil {
		return nil, errors.Wrapf(err, "create access token credential %s", credentialID)
	}

	// create 失败（含蓝盾创建失败、本地落库失败）时回滚刚建的凭证；
	// 若蓝盾流水线已创建，create 内部会先删远程流水线，避免留下孤儿实例。
	// 凭证删除失败只告警，错误仍返回 createErr
	pipeline, createErr := m.create(ctx, client, store, project, appID, pipelineType, credentialID, callbackURL)
	if createErr != nil {
		if delErr := client.DeleteCredential(ctx, project.Code, credentialID); delErr != nil {
			log.Warnf(
				ctx, "rollback build-trigger credential %s in project %s failed after pipeline create error: %v",
				credentialID, project.Code, delErr,
			)
		}
		return nil, createErr
	}
	return pipeline, nil
}

// Cleanup 清理指定应用的触发专用流水线与回调凭证（幂等）
//
// 策略管理侧应在应用最后一条触发策略删除后同步调用；本方法不查询 PolicyStore，
// 是否该删由调用方保证。删除顺序：蓝盾流水线 → 回调凭证 → 本地记录；
// 蓝盾删流水线失败时保留本地并返回错误；凭证回收失败则告警并继续清本地
func (m *TriggerPipelineManager) Cleanup(ctx context.Context, appID string) error {
	pipelineType := string(BuildTriggerPipelineType(appID))
	store, err := NewPipelineStoreMongo(database.Client(), database.Name())
	if err != nil {
		return errors.Wrap(err, "create pipeline store")
	}
	pipeline, err := store.GetByWorkspaceAndType(ctx, m.workspaceID, pipelineType)
	if err != nil {
		// 本地已无记录视为幂等成功（可能已被清理或从未 Ensure）
		if errors.Is(err, ErrPipelineNotFound) {
			return nil
		}
		return errors.Wrapf(err, "get workspace %s pipeline %s", m.workspaceID, pipelineType)
	}

	client, err := cloudbkci.New(auth.MustGetUser(ctx))
	if err != nil {
		return errors.Wrap(err, "create bkci client")
	}

	// 远程已不存在可继续；其它错误中止，避免本地被清掉后失去对账依据
	if err = client.DeletePipeline(ctx, pipeline.ProjectCode, pipeline.ID); err != nil {
		if !errors.Is(err, cloudbkci.ObjectNotFound) {
			return errors.Wrapf(err, "delete bkci pipeline %s in project %s", pipeline.ID, pipeline.ProjectCode)
		}
	}

	// 优先用落库的 credentialID；旧数据缺失时按 appID 重算（与 Ensure 生成规则一致）
	credentialID := pipeline.CallbackCredentialID
	if credentialID == "" {
		credentialID = m.buildCredentialID(appID)
	}
	if err = client.DeleteCredential(ctx, pipeline.ProjectCode, credentialID); err != nil {
		if !errors.Is(err, cloudbkci.ObjectNotFound) {
			log.Warnf(
				ctx, "delete build-trigger credential %s in project %s failed, continue clearing local record: %v",
				credentialID, pipeline.ProjectCode, err,
			)
		}
	}

	// 本地 Delete 对不存在静默成功
	if err = store.Delete(ctx, m.workspaceID, pipelineType); err != nil {
		return errors.Wrapf(err, "delete local pipeline %s", pipelineType)
	}
	return nil
}

// create 渲染模板实例字段、创建蓝盾流水线，并写入本地 bkci_pipelines
//
// 调用方须已创建回调凭证；本方法失败时由 Ensure 负责回滚凭证。
// 若蓝盾 CreatePipeline 已成功但本地落库失败，本方法会尽力 DeletePipeline，
// 避免蓝盾侧留下无本地记录、Cleanup 也无法回收的孤儿流水线（删失败只告警）。
// name/stages 经 [[ ]] 注入 appID、callbackURL、credentialID；description 直接用模板文案
func (m *TriggerPipelineManager) create(
	ctx context.Context,
	client cloudbkci.Client,
	store PipelineStore,
	project *Project,
	appID, pipelineType, credentialID, callbackURL string,
) (*Pipeline, error) {
	// 查模板类型为 build-trigger（非复合 type）；复用 PipelineManager 的加载逻辑
	tmpl, err := NewPipelineManager(m.workspaceID).getPipelineTemplate(
		ctx, string(PipelineTypeBuildTrigger),
	)
	if err != nil {
		return nil, errors.Wrap(err, "get build-trigger pipeline template")
	}

	renderCtx := map[string]any{
		pipelineTmplCtxKeyAppID:        appID,
		pipelineTmplCtxKeyCallbackURL:  callbackURL,
		pipelineTmplCtxKeyCredentialID: credentialID,
	}
	name, err := renderBuildTriggerText(tmpl.Name, renderCtx)
	if err != nil {
		return nil, err
	}
	stages, err := renderBuildTriggerStages(tmpl.Stages, renderCtx)
	if err != nil {
		return nil, err
	}

	pipelineID, err := client.CreatePipeline(ctx, project.Code, name, tmpl.Description, stages)
	if err != nil {
		return nil, errors.Wrapf(err, "create bkci build-trigger pipeline for app %s", appID)
	}

	pipeline := &Pipeline{
		ID:                   pipelineID,
		Type:                 pipelineType,
		WorkspaceID:          m.workspaceID,
		ProjectCode:          project.Code,
		Name:                 name,
		Description:          tmpl.Description,
		TemplateVersion:      tmpl.Version,
		CallbackCredentialID: credentialID,
		Creator:              auth.MustGetUser(ctx).ID,
	}
	if err = store.Create(ctx, pipeline); err != nil {
		// 本地未落库则远程也必须回收，否则下次 Ensure 会再建一条且 Cleanup 对不上账
		if delErr := client.DeletePipeline(ctx, project.Code, pipelineID); delErr != nil &&
			!errors.Is(delErr, cloudbkci.ObjectNotFound) {
			log.Warnf(
				ctx, "rollback build-trigger pipeline %s in project %s failed after local insert error: %v",
				pipelineID, project.Code, delErr,
			)
		}
		return nil, errors.Wrap(err, "insert build-trigger pipeline to db")
	}
	return pipeline, nil
}

// buildCallbackURL 从 httpServer.publicBaseURL 拼装应用触发回调完整 URL
//
// 基址为独立配置，不能从既有 gateway/host 推导；去尾斜杠后与固定 path 拼接，
// appID 做 PathEscape，避免特殊字符破坏路径；基址必须是仅含 scheme + host(+port) 的
// http/https URL，拒绝 path / query / fragment，以免拼出错误回调地址
func (m *TriggerPipelineManager) buildCallbackURL(appID string) (string, error) {
	base := ""
	if config.G != nil {
		base = strings.TrimRight(strings.TrimSpace(config.G.HTTPServer.PublicBaseURL), "/")
	}
	if base == "" {
		return "", errors.New("httpServer.publicBaseURL is required to ensure build-trigger pipeline")
	}
	parsed, err := url.ParseRequestURI(base)
	if err != nil {
		return "", errors.Wrap(err, "invalid httpServer.publicBaseURL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("httpServer.publicBaseURL must be http or https")
	}
	if parsed.Host == "" {
		return "", errors.New("httpServer.publicBaseURL must include a host")
	}
	if strings.Trim(parsed.Path, "/") != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("httpServer.publicBaseURL must not include path, query, or fragment")
	}
	return base + fmt.Sprintf(buildTriggerCallbackPathTmpl, url.PathEscape(appID)), nil
}

// buildCredentialID 生成稳定可回收的蓝盾凭证 ID
//
// 规则：前缀 + 净化后的 appID（仅字母、数字、下划线，连字符换成下划线）；
// 未超长时保留可读形式；超长时用 appID 的 sha256 摘要填满剩余长度，避免同前缀截断冲突。
// 净化后为空则退回 bkms_bt；同 appID 多次 Ensure/Cleanup 命中同一 ID
func (m *TriggerPipelineManager) buildCredentialID(appID string) string {
	sanitized := nonCredentialIDCharPattern.ReplaceAllString(appID, "_")
	sanitized = strings.Trim(sanitized, "_")
	if sanitized == "" {
		return strings.TrimRight(buildTriggerCredentialIDPrefix, "_")
	}
	id := buildTriggerCredentialIDPrefix + sanitized
	if len(id) <= buildTriggerCredentialIDMaxLen {
		return id
	}
	return buildTriggerCredentialIDPrefix + credentialIDDigest(appID)
}

// credentialIDDigest 取 appID 的 sha256 十六进制前缀，长度恰好填满凭证 ID 上限
func credentialIDDigest(appID string) string {
	digestLen := buildTriggerCredentialIDMaxLen - len(buildTriggerCredentialIDPrefix)
	sum := sha256.Sum256([]byte(appID))
	return hex.EncodeToString(sum[:digestLen/2])
}

// generateCallbackToken 生成回调鉴权 token（hex 编码的加密随机数，仅创建凭证时使用一次）
func (m *TriggerPipelineManager) generateCallbackToken() (string, error) {
	buf := make([]byte, buildTriggerTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// renderBuildTriggerText 渲染触发流水线模板中的文本字段（如带 appID 的显示名）
func renderBuildTriggerText(text string, renderCtx map[string]any) (string, error) {
	rendered, err := renderPipelineTemplate(buildTriggerRenderName, []byte(text), renderCtx)
	if err != nil {
		return "", err
	}
	return string(rendered), nil
}

// renderBuildTriggerStages 深拷贝模板 stages，并用 [[ ]] 注入实例期字段
//
// 经 JSON round-trip 避免修改入参；占位符在资产中自逃逸，Reload 后仍为 [[ .callbackURL ]] 等形式
func renderBuildTriggerStages(stages []map[string]any, renderCtx map[string]any) ([]map[string]any, error) {
	raw, err := json.Marshal(stages)
	if err != nil {
		return nil, errors.Wrap(err, "marshal pipeline template stages")
	}
	renderedRaw, err := renderPipelineTemplate(buildTriggerRenderStages, raw, renderCtx)
	if err != nil {
		return nil, err
	}

	var rendered []map[string]any
	if err = json.Unmarshal(renderedRaw, &rendered); err != nil {
		return nil, errors.Wrap(err, "unmarshal rendered pipeline stages")
	}
	return rendered, nil
}
