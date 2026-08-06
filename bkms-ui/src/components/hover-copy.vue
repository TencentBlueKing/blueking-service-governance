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

<template>
  <div class="w-full flex items-center gap-[4px] content-hover">
    <div
      v-bk-tooltips="{ content: tooltip, disabled: !tooltip }"
      class="max-w-[calc(100%-20px)]"
      :class="{ 'border-b border-dashed border-[#979ba5] cursor-pointer': tooltip }"
    >
      <OverflowTitle :type="tooltip ? undefined : 'tips'">
        {{ text || emptyPlaceholder }}
      </OverflowTitle>
    </div>
    <slot />
    <Copy
      v-if="copyValue"
      class="cursor-pointer content-item hover:text-[#3A84FF]"
      fill="#3a84ff"
      height="16"
      :title="$t('复制')"
      width="16"
      @click.stop="copyText(copyValue)"
    />
  </div>
</template>

<script lang="ts" setup>
  import { OverflowTitle } from 'bkui-vue';
  import { Copy } from 'bkui-vue/lib/icon';
  import { copyText } from '~/common/util';
  interface IProps {
    copyValue: string;
    emptyPlaceholder?: string;
    text: number | string;
    tooltip?: string;
  }

  withDefaults(defineProps<IProps>(), {
    emptyPlaceholder: '--',
  });
</script>
