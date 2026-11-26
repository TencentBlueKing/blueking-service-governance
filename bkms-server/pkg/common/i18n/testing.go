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

package i18n

import "testing/fstest"

// UseCatalogForTest 用给定的翻译文件内容（语言 -> .po 文件内容）替换当前生效的翻译，返回恢复函数。
// 仅供其他包的测试构造内嵌翻译文件中不存在的状态（如译文为空、fuzzy），不能在并行测试中使用
func UseCatalogForTest(files map[Lang]string) (restore func(), err error) {
	fsys := fstest.MapFS{}
	for lang, body := range files {
		fsys["locales/"+string(lang)+".po"] = &fstest.MapFile{Data: []byte(body)}
	}
	c, err := load(fsys)
	if err != nil {
		return nil, err
	}
	prev := messages
	messages = c
	return func() { messages = prev }, nil
}
