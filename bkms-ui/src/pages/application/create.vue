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
  <MsHeader
    :title="$t('创建应用')"
    :trigger-back="goBack"
  >
    <template v-if="curStep < 4">
      <div class="flex-1 flex items-center justify-center">
        <Steps
          class="max-w-[460px]"
          :cur-step="curStep"
          line-type="solid"
          :steps="stepsData"
        />
      </div>
    </template>
  </MsHeader>
  <RouterView
    v-slot="{ Component }"
    class="main w-full overflow-auto"
  >
    <component
      :is="Component"
      :step="curStep"
      @next="next"
      @pre="pre"
      @update-step="updateStep"
      @update-text="updateText"
    />
  </RouterView>
</template>
<script lang="ts" setup>
  import { onMounted, ref } from 'vue';

  import { Steps } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import MsHeader from '~/components/ms-header.vue';
  import { APP_TYPES, IAppType } from '~/composables/app-type';

  const { t } = useI18n();

  // 路由名 → 应用类型 映射表（集中维护，新增类型只需在此添加）
  const ROUTE_NAME_TO_APP_TYPE: Record<string, IAppType> = {
    createTrpcTemplateApp: APP_TYPES.TRPC,
    createHelmTemplateApp: APP_TYPES.HELM,
    createTAFTemplateApp: APP_TYPES.TAF,
    createAgonesTemplateApp: APP_TYPES.AGONES,
  };

  // 步骤条数据配置
  const STEPS_CONFIG: Record<string, { title: string }[]> = {
    default: [{ title: t('选择模板') }, { title: t('参数配置') }, { title: t('创建应用') }],
    [APP_TYPES.HELM]: [{ title: t('选择模板') }, { title: t('参数配置') }],
    [APP_TYPES.AGONES]: [{ title: t('选择模板') }, { title: t('参数配置') }],
  };

  const curStep = ref(1);
  const stepsData = ref(STEPS_CONFIG.default.map(step => ({ title: t(step.title) })));

  function next() {
    curStep.value += 1;
  }
  function pre() {
    curStep.value -= 1;
    // router.back();
  }
  function updateStep(step: number) {
    curStep.value = step;
  }
  function updateText(app: { type: string }) {
    stepsData.value = (STEPS_CONFIG[app.type] || STEPS_CONFIG.default).map(step => ({ title: t(step.title) }));
  }

  const route = useRoute(); // 获取路由信息
  const router = useRouter();
  function goBack() {
    // 显式指定 fallback：resolveParent 推导到模板选择页(createApplication)，但业务上应返回应用列表页(app)
    router.back({ name: 'app', params: { space: route.params.space } });
  }

  // 初始化步骤数据，处理页面刷新的情况
  function initStepsData() {
    const appType = ROUTE_NAME_TO_APP_TYPE[route.name as string] || '';
    stepsData.value = (STEPS_CONFIG[appType] || STEPS_CONFIG.default).map(step => ({ title: t(step.title) }));
  }

  onMounted(() => {
    initStepsData();
  });
</script>
<style lang="postcss" scoped>
  .main {
    height: calc(100% - 52px);
  }
</style>
