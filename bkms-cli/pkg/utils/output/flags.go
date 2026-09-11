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

package output

import "github.com/spf13/cobra"

// formatValue 在参数解析时校验输出格式，仅将合法值写入命令绑定的变量。
type formatValue struct {
	value *string
}

func (v *formatValue) Set(raw string) error {
	if err := ValidateFormat(raw); err != nil {
		return err
	}
	*v.value = raw
	return nil
}

func (v *formatValue) String() string {
	if v.value == nil {
		return ""
	}
	return *v.value
}

func (v *formatValue) Type() string {
	return "format"
}

// AddFormatFlag 注册 --output/-o，输出格式及 JQ 表达式在参数解析时完成静态校验。
func AddFormatFlag(cmd *cobra.Command, value *string) {
	*value = ""
	cmd.Flags().VarP(&formatValue{value: value}, "output", "o", FlagUsage)
}
