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
  <div
    v-if="title"
    class="flex items-center h-[52px] shadow px-[24px] bg-[#fff]"
  >
    <span
      class="bkms-icon bkms-icon-return-small cursor-pointer text-[36px]"
      :style="{ color: backColor }"
      @click="handleBack"
    >
    </span>
    <span class="text-[16px]">{{ title }}</span>
    <slot></slot>
  </div>
</template>
<script lang="ts" setup>
  import { useRouter } from 'vue-router';

  interface Props {
    back?: boolean;
    backColor?: string;
    title?: string;
    triggerBack?: () => void;
  }

  const props = withDefaults(defineProps<Props>(), {
    backColor: '#3a84ff',
  });

  const router = useRouter();
  function handleBack() {
    if (props?.triggerBack) {
      props.triggerBack();
    } else {
      router.back();
    }
  }
</script>
