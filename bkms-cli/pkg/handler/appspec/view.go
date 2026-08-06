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

package appspec

import (
	"context"
	"fmt"
	"sync"

	"github.com/pkg/errors"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/client"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-cli/pkg/utils/output"
)

// ViewHandler 查询并输出指定 section 配置，支持多种输出格式。
func ViewHandler(
	ctx context.Context,
	appID, envName string,
	section client.AppSpecSectionName,
	outputFormat string,
) error {
	data, err := viewSection(ctx, appID, envName, section)
	if err != nil {
		return errors.Wrapf(err, "view %s", section)
	}

	if outputFormat == "" {
		fmt.Println(FormatSectionTable(section, data))
		return nil
	}

	formatted, err := output.FormatData(ctx, data, outputFormat)
	if err != nil {
		return errors.Wrap(err, "format output")
	}
	fmt.Println(formatted)
	return nil
}

// ViewAllHandler 查询并输出所有 section 配置。
func ViewAllHandler(ctx context.Context, appID, envName, outputFormat string) error {
	result, err := viewAll(ctx, appID, envName)
	if err != nil {
		return errors.Wrap(err, "view all sections")
	}

	if outputFormat == "" {
		fmt.Println(FormatViewAllTable(result))
		return nil
	}

	formatted, err := output.FormatData(ctx, result, outputFormat)
	if err != nil {
		return errors.Wrap(err, "format output")
	}
	fmt.Println(formatted)
	return nil
}

// ViewStartCommandHandler 查询应用启动命令配置。
func ViewStartCommandHandler(ctx context.Context, appID string) (*StartCommandOutput, error) {
	cli := client.New()
	app, err := cli.GetAppDetail(ctx, appID)
	if err != nil {
		return nil, err
	}

	out := new(StartCommandOutput)
	if app.AppModelSpec != nil {
		out.Command = app.AppModelSpec.Command
		out.Args = app.AppModelSpec.Args
	}
	return out, nil
}

// --- internal helpers ---

func viewSection(ctx context.Context, appID, envName string, section client.AppSpecSectionName) (any, error) {
	cli := client.New()
	return viewSectionWith(ctx, cli, appID, envName, section)
}

func viewSectionWith(
	ctx context.Context,
	cli client.Client,
	appID, envName string,
	section client.AppSpecSectionName,
) (any, error) {
	switch section {
	case client.AppSpecSectionResources:
		return viewSectionTyped[client.ResourcesConfig](ctx, cli, appID, envName, section)
	case client.AppSpecSectionUpdateStrategy:
		return viewSectionTyped[client.UpdateStrategyConfig](ctx, cli, appID, envName, section)
	case client.AppSpecSectionLifecycle:
		v, err := viewSectionTyped[client.LifecycleConfig](ctx, cli, appID, envName, section)
		if err != nil {
			return nil, err
		}
		v.Sanitize()
		return v, nil
	case client.AppSpecSectionProbe:
		return viewSectionTyped[client.ProbeConfig](ctx, cli, appID, envName, section)
	case client.AppSpecSectionLabels:
		v, err := viewSectionTyped[client.LabelsConfig](ctx, cli, appID, envName, section)
		if err != nil {
			return nil, err
		}
		if v.IsEmpty() {
			return nil, nil //nolint:nilnil // 空配置时返回 nil 表示无数据
		}
		return v, nil
	case client.AppSpecSectionAnnotations:
		v, err := viewSectionTyped[client.AnnotationsConfig](ctx, cli, appID, envName, section)
		if err != nil {
			return nil, err
		}
		if v.IsEmpty() {
			return nil, nil //nolint:nilnil // 空配置时返回 nil 表示无数据
		}
		return v, nil
	default:
		return nil, nil //nolint:nilnil // 未知 section 返回 nil 表示无数据
	}
}

func viewSectionTyped[T any](
	ctx context.Context,
	cli client.Client,
	appID, envName string,
	section client.AppSpecSectionName,
) (*T, error) {
	result := new(T)
	var err error
	if envName == "" {
		err = cli.GetAppSpecDefaultSection(ctx, appID, section, result)
	} else {
		err = cli.GetAppSpecEnvEffectiveSection(ctx, appID, envName, section, result)
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

func viewAll(ctx context.Context, appID, envName string) (*ViewAllResult, error) {
	result := &ViewAllResult{}
	var mu sync.Mutex
	var wg sync.WaitGroup

	cli := client.New()

	wg.Add(1)
	go func() {
		defer wg.Done()
		out, err := ViewStartCommandHandler(ctx, appID)
		if err != nil {
			return
		}
		mu.Lock()
		result.StartCommand = out
		mu.Unlock()
	}()

	sections := []client.AppSpecSectionName{
		client.AppSpecSectionLifecycle,
		client.AppSpecSectionProbe,
		client.AppSpecSectionResources,
		client.AppSpecSectionUpdateStrategy,
		client.AppSpecSectionLabels,
		client.AppSpecSectionAnnotations,
	}

	for _, section := range sections {
		wg.Add(1)
		go func(s client.AppSpecSectionName) {
			defer wg.Done()
			data, err := viewSectionWith(ctx, cli, appID, envName, s)
			if err != nil || data == nil {
				return
			}
			mu.Lock()
			result.setSection(s, data)
			mu.Unlock()
		}(section)
	}

	wg.Wait()
	return result, nil
}
