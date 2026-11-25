<!--
 - TencentBlueKing is pleased to support the open source community by making
 - 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 - Copyright (C) Tencent. All rights reserved.
 - Licensed under the MIT License (the "License"); you may not use this file except
 - in compliance with the License. You may obtain a copy of the License at
 -
 -  http://opensource.org/licenses/MIT
 -
 - Unless required by applicable law or agreed to in writing, software distributed
 - under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
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
    <div>
      <p
        v-if="platformConfig.i18n.footerInfoHTML"
        v-bk-xss-html="platformConfig.i18n.footerInfoHTML"
        class="mb-[8px]"
      ></p>
      <p class="text-[#979ba5]">
        {{ platformConfig.footerCopyrightContent }}
      </p>
    </div>
  </footer>
</template>
<script setup lang="ts">
  import usePlatform from '~/composables/use-platform';

  const { platformConfig } = usePlatform();
</script>
<style lang="postcss" scoped>
  /* platform-config 生成的链接 <a class="link-item"> 无内置样式 */
  :deep(.link-item) {
    color: #3a84ff;

    &:hover {
      color: #699df4;
    }
  }
</style>
