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
    class="flex items-center justify-center mx-[100px] my-[24px] rounded-[8px] bg-[#FFF] shadow-[0_2px_4px_0_#1919290d] min-h-[calc(100vh-150px)]"
  >
    <div class="flex flex-col items-center">
      <img
        alt="home"
        class="w-[80px] h-[80px] mb-[16px] rounded-[20px]"
        src="@/assets/home-platform-logo.png"
      />

      <h1 class="text-[24px] font-bold text-[#313238] mb-[16px]">{{ $t('欢迎使用蓝鲸服务治理平台') }}</h1>
      <p class="text-[14px] text-[#4D4F56]">
        {{ $t('一站式应用全生命周期管理服务，从开发、构建、部署到运维的完整解决方案') }}
      </p>

      <div class="grid grid-cols-3 gap-[16px] my-[24px] w-[932px]">
        <div
          v-for="feature in features"
          :key="feature.title"
          class="bg-[#ffffff66] rounded-[16px] p-[24px] shadow-[0_8px_32px_0_#1f26870d] hover:shadow-[0_4px_12px_0_rgba(0,0,0,0.08)] transition-shadow duration-200"
        >
          <!-- 特性图标预留位 -->
          <div
            class="w-[44px] h-[44px] rounded-[10px] flex items-center justify-center mb-[12px]"
            :style="{ background: feature.bgGradient }"
          >
            <i
              class="text-[22px] text-[#fff]"
              :class="feature.icon"
            />
          </div>
          <div class="text-[14px] font-bold text-[#313238] mb-[8px]">{{ feature.title }}</div>
          <div class="text-[12px] text-[#4D4F56] leading-[20px]">{{ feature.desc }}</div>
        </div>
      </div>

      <div class="flex items-center gap-[8px]">
        <Button
          theme="primary"
          @click="emit('create')"
        >
          <Plus
            :height="24"
            :width="24"
          />
          {{ $t('新建空间') }}
        </Button>
        <Button @click="emit('guide')">{{ $t('查看接入指引') }}</Button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { Button } from 'bkui-vue';
  import { Plus } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';

  const { t } = useI18n();

  interface Emits {
    (e: 'create'): void;
    (e: 'guide'): void;
  }

  const emit = defineEmits<Emits>();

  const features = [
    {
      icon: 'bkms-icon bkms-icon-auto-refresh-line',
      bgGradient: 'linear-gradient(180deg, #76AAFF 0%, #3A84FF 100%)',
      title: t('全流程自动化'),
      desc: t('打通代码构建、部署到运维观测的完整闭环'),
    },
    {
      icon: 'bkms-icon bkms-icon-jicheng',
      bgGradient: 'linear-gradient(135deg, #10B981 0%, #34D399 100%)',
      title: t('深度生态集成'),
      desc: t('集成配置管理、服务发现、全链路追踪'),
    },
    {
      icon: 'bkms-icon bkms-icon-shipei',
      bgGradient: 'linear-gradient(135deg, #8B5CF6 0%, #A78BFA 100%)',
      title: t('多场景适配'),
      desc: t('支持 tRPC、Helm、TAF 等多种应用类型'),
    },
  ];
</script>
