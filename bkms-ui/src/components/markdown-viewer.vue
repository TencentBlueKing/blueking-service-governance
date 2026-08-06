<!--
 - TencentBlueKing is pleased to support the open source community by making
 - 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 - Copyright (C) Tencent. All rights reserved.
 - Licensed under the MIT License (the "License"); you may not use this file except
 - in compliance with the License. You may obtain a copy of the License at
 -
 -  http://opensource.org/licenses/MIT
 -
 - Unless required by applicable law or agreed to in writing, software distributed under
 - the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 - either express or implied. See the License for the specific language governing permissions and
 - limitations under the License.
 -
 - We undertake not to change the open source license (MIT license) applicable
 - to the current version of the project delivered to anyone in the future.
-->

<!-- eslint-disable vue/no-v-html -->
<template>
  <div
    v-bk-xss-html="markdownToHtml"
    class="markdown-body p-[16px]"
  ></div>
</template>
<script lang="ts" setup>
  import { computed } from 'vue';

  import hljs from 'highlight.js/lib/core';
  import bash from 'highlight.js/lib/languages/bash';
  import go from 'highlight.js/lib/languages/go';
  import plaintext from 'highlight.js/lib/languages/plaintext';
  import protobuf from 'highlight.js/lib/languages/protobuf';
  import shell from 'highlight.js/lib/languages/shell';
  import yaml from 'highlight.js/lib/languages/yaml';
  import { Marked } from 'marked';
  import { markedHighlight } from 'marked-highlight';

  import 'github-markdown-css'; // 整体 markdown 样式

  import 'highlight.js/styles/github.css'; // 代码块高亮样式

  const props = defineProps({
    value: { type: String, default: '' },
  });

  hljs.registerLanguage('yaml', yaml);
  hljs.registerLanguage('bash', bash);
  hljs.registerLanguage('go', go);
  hljs.registerLanguage('shell', shell);
  hljs.registerLanguage('protobuf', protobuf);
  hljs.registerLanguage('plaintext', plaintext);

  // markdown 转 html
  const marked = new Marked(
    markedHighlight({
      emptyLangClass: 'hljs',
      langPrefix: 'hljs language-',
      highlight(code, lang) {
        const language = hljs.getLanguage(lang) ? lang : 'plaintext';
        return hljs.highlight(code, { language }).value;
      },
    }),
  );

  const markdownToHtml = computed(() => marked.parse(props.value, { async: false }));
</script>
