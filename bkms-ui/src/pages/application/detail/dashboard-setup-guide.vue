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
  <i18n-t
    class="text-[#4D4F56]"
    :class="compact ? 'text-[12px] leading-[20px]' : 'text-[14px] leading-[22px]'"
    keypath="可嵌入蓝鲸监控仪表盘，查看应用指标。配置步骤见 {0}"
    tag="span"
  >
    <a
      class="text-[#3A84FF]"
      :href="docUrl"
      rel="noopener noreferrer"
      target="_blank"
    >
      {{ $t('参考文档') }}
      <Share class="ml-[2px] align-[-2px]" />
    </a>
  </i18n-t>
</template>

<script setup lang="ts">
  import { computed } from 'vue';

  import { Share } from 'bkui-vue/lib/icon';
  import { DOC_LINKS } from '~/common/const';

  withDefaults(
    defineProps<{
      compact?: boolean;
    }>(),
    {
      compact: false,
    },
  );

  const docUrl = computed(() => `${import.meta.env.BK_DOC_URL}${DOC_LINKS.DASHBOARD_GUIDE}`);
</script>
