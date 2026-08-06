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
  <!-- trpc 环境变量提示 -->
  <Alert
    closable
    theme="info"
  >
    <div class="flex flex-col text-[12px]">
      <i18n-t
        keypath="推荐用法：点击复制获取 {0}，由 tRPC 框架运行时解析；平台不会将环境变量值渲染进配置文件。"
        tag="div"
      >
        <span class="mx-[4px] font-bold text-[#313238]">${Key}</span>
      </i18n-t>
      <i18n-t
        keypath="特殊场景：若需平台在下发配置前完成变量渲染，可通过复制按钮旁的下拉菜单获取 {0}"
        tag="div"
      >
        <span
          class="mx-[4px] font-bold text-[#313238]"
          v-text="'${{ env.Key }}'"
        ></span>
      </i18n-t>
    </div>
  </Alert>
</template>

<script setup lang="ts">
  import { Alert } from 'bkui-vue';
</script>
