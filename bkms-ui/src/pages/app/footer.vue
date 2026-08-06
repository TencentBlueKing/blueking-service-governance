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
  <footer
    class="w-full h-[50px] text-center text-[12px] leading-[12px] flex flex-col justify-center absolute bottom-0 border-t border-t-solid border-t-[#DCDEE5]"
  >
    <!-- <div v-if="platformConfig.i18n.footerInfoHTML">
      <p class="mb-[8px] text-[#3A84FF]" v-html="platformConfig.i18n.footerInfoHTML"></p>
      <p class="text-[#979ba5]">
          {{ platformConfig.footerCopyrightContent }}
      </p>
    </div> -->
    <ul class="flex items-center justify-center mb-[8px]">
      <li
        v-for="(item, index) in links"
        :key="index"
      >
        <a
          class="text-[#3A84FF]"
          :href="item.href"
          :target="item.target"
        >
          {{ item.name }}
        </a>
        <span
          v-if="index < 1"
          class="mx-[5px]"
          >|</span
        >
      </li>
    </ul>
    <p class="text-[#979ba5]">{{ Copyright }}</p>
  </footer>
</template>
<script setup lang="ts">
  import { computed, ref } from 'vue';

  import { useI18n } from 'vue-i18n';

  const { t } = useI18n();
  const links = [
    {
      name: t('联系BK助手'),
      href: 'https://bk.tencent.com/s-mart/community',
      target: '_blank',
    },
    {
      name: t('蓝鲸桌面'),
      href: 'https://bk.tencent.com/s-mart/desktop',
      target: '_blank',
    },
  ];
  const version = ref('V1.1.1');
  const Copyright = computed(() => `Copyright © 2012 Tencent BlueKing. All Rights Reserved. ${version.value}`);
</script>
