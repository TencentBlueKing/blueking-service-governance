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
  <div class="flex flex-col h-full">
    <FlexRow class="px-[12px] py-[8px] bg-[#EAEBF0] shadow-[0_2px_4px_0_#0000001a] mb-[16px]">
      <template #left>
        <div class="flex items-center">
          <EnvSelectPanel
            v-model="curEnv"
            v-model:item="curEnvItem"
            class="w-[240px] mr-[16px]"
            init-first-env-when-empty
            type="helm"
          />
          <!-- 泳道 -->
          <LaneSelect
            v-if="laneList.length"
            v-model="curLaneName"
            :list="laneList"
            type="helm"
          />
          <div
            v-if="deployHistoryList.length"
            class="flex items-center border border-[#c4c6cc] bg-[#fff]"
          >
            <span class="px-[8px] py-[6px] border-r border-r-[#c4c6cc]">{{ $t('部署状态') }}</span>
            <StatusIcon
              class="px-[8px]"
              :status="latestDeployStatus"
              :status-color-map="statusColorMap"
              :status-text-map="statusTextMap"
            />
          </div>
        </div>
      </template>
      <template
        v-if="deployHistoryList.length"
        #right
      >
        <div class="flex items-center gap-[12px]">
          <Button
            v-bk-tooltips="{
              content: updateButtonDisabledTip,
              disabled: !isUpdateButtonDisabled,
            }"
            :disabled="isUpdateButtonDisabled"
            theme="primary"
            @click="handleShowDeployDialog('Recreate')"
            >{{ $t('更新') }}
          </Button>
          <Popover
            ref="moreOperationsRef"
            ext-cls="more-operations"
            :offset="12"
            placement="bottom"
            theme="light"
          >
            <Ellipsis
              class="transform-rotate-90 w-[24px] h-[24px] p-[3px] rounded-full cursor-pointer text-[#4D4F56] text-[20px] hover:bg-gray-200"
            />
            <template #content>
              <div
                class="px-[12px] py-[6px] cursor-pointer hover:bg-[#F0F1F5] hover:text-[#3A84FF]"
                @click="handleRemoveDeploy"
              >
                {{ $t('移除部署') }}
              </div>
            </template>
          </Popover>
        </div>
      </template>
    </FlexRow>
    <!-- 存在部署的应用 -->
    <Tab
      v-model:active="activeTab"
      class="flex-1 overflow-hidden"
      :label-height="40"
      type="unborder-card"
    >
      <Tab.TabPanel
        :name="TAB_NAMES.topo"
        render-directive="if"
      >
        <template #label>
          {{ $t('资源拓扑') }}
        </template>
        <ResourceTopology :env-name="curEnv">
          <template #empty-deploy>
            <Button
              class="mt-[8px]"
              theme="primary"
              @click="handleShowDeployDialog('RollingUpdate')"
            >
              {{ $t('立即部署') }}
            </Button>
          </template>
        </ResourceTopology>
      </Tab.TabPanel>
      <Tab.TabPanel
        :name="TAB_NAMES.history"
        render-directive="show"
      >
        <template #label>
          {{ $t('部署历史') }}
        </template>
        <!-- 空状态 -->
        <Exception
          v-if="!deployHistoryLoading && !deployHistoryList.length && !hasSearchValue"
          class="large-exception"
          scene="part"
          type="empty"
        >
          <template #type>
            <img src="/empty.svg" />
          </template>
          <template #description>
            <div class="text-[#4D4F56] text-[14px] leading-[22px]">{{ $t('该环境尚未部署应用') }}</div>
          </template>
          <Button
            theme="primary"
            @click="handleShowDeployDialog('RollingUpdate')"
            >{{ $t('立即部署') }}</Button
          >
        </Exception>
        <deployHistory
          v-else
          :env-name="curEnv"
          :initial-loading="deployHistoryLoading"
          :lane-name="curLaneName"
          :skip-initial-fetch="skipInitialFetch"
          @search-change="search => (hasSearchValue = search.trim() !== '')"
        />
      </Tab.TabPanel>
    </Tab>
    <!-- 部署/更新 -->
    <DeployApplication
      v-model:is-show="isShowDeploy"
      :deploy-type="curDeployType"
      :env-item="curEnvItem"
      :lane-name="curLaneName"
      @deploy="getDeployHistories"
    />
  </div>
</template>
<script setup lang="ts">
  import { computed, h, nextTick, ref, watch } from 'vue';

  import { Button, Exception, InfoBox, Message, Popover, Tab } from 'bkui-vue';
  import { Ellipsis } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { DeleteHelmDeployRequest } from '~/@types/v1/deploy';
  import { EnvOutput } from '~/@types/v1/env';
  import { DeployService } from '~/api/modules/v1';
  import { isHelmLikeAppType } from '~/composables/app-type';
  import { useUrlQuerySync } from '~/composables/use-url-query-sync';
  import { useAppDetail } from '~/stores/app-detail';

  import ResourceTopology from '../../components/topo/index.vue';
  import DeployApplication from './deploy-application.vue';
  import deployHistory from './deploy-history.vue';
  import removeInfoxBox from './remove-infoxBox.vue';
  import { useHelmDeploy } from './use-helm-deploy';

  const { t } = useI18n();
  const appDetailStore = useAppDetail();
  // 部署应用
  const {
    laneList,
    deployHistoryList,
    latestDeployStatus,
    statusColorMap,
    statusTextMap,
    handleListDeployHistories,
    handleGetChartList,
    handleGetLanesList,
  } = useHelmDeploy();

  const curEnv = ref('');
  const curEnvItem = ref<EnvOutput>();
  const deployHistoryLoading = ref(false);
  const skipInitialFetch = ref(false);
  const hasSearchValue = ref(false);
  const isShowDeploy = ref(false);
  const curDeployType = ref<'Recreate' | 'RollingUpdate'>('Recreate');

  // Tab 名称常量（模板与校验同源）
  const TAB_NAMES = {
    topo: 'topo',
    history: 'history',
  } as const;

  // Tab 与 URL query（activeTab）双向同步锚定
  // env 参数与当前环境双向同步（环境列表异步加载，不配置 allowed 直接透传；区别于一次性定位参数 envName）
  const { fields } = useUrlQuerySync({
    activeTab: {
      queryKey: 'activeTab',
      data: {
        allowed: Object.values(TAB_NAMES),
        default: TAB_NAMES.topo,
      },
    },
    env: {
      queryKey: 'env',
      data: { default: '' },
    },
  });
  const activeTab = fields.activeTab;
  const targetEnvName = fields.env;

  // URL 中的 envName → 初始化当前环境（首次进入时生效）
  // curEnv → 写回 URL（首次默认环境与用户切换环境都写入，便于分享直达）
  const isInitializingEnvFromUrl = ref(false);

  // 泳道
  const curLaneName = ref('');
  const moreOperationsRef = ref<InstanceType<typeof Popover>>();

  const isUpdateButtonDisabled = computed(() => {
    const status = latestDeployStatus.value;
    return ['pending-upgrade', 'pending-install', 'pending-rollback', 'uninstalling'].includes(status);
  });

  // 禁用提示
  const updateButtonDisabledTip = computed(() => {
    const status = latestDeployStatus.value;
    const statusTips: Record<string, string> = {
      'pending-upgrade': t('部署进行中，请勿重复部署'),
      'pending-install': t('部署进行中，请勿重复部署'),
      'pending-rollback': t('回滚中，请稍后再试'),
      uninstalling: t('卸载中，请稍后再试'),
    };
    return statusTips[status] || '';
  });

  // 获取部署历史
  async function getDeployHistories() {
    if (!appDetailStore.appID) return;
    deployHistoryLoading.value = true;
    skipInitialFetch.value = true; // 标记跳过子组件初始化获取
    await handleListDeployHistories({
      page: 1,
      pageSize: 10,
      envName: curEnv.value,
      trafficLaneName: curLaneName.value,
    });
    deployHistoryLoading.value = false;
  }

  // 移除部署
  async function handleRemoveDeploy() {
    moreOperationsRef.value?.hide();
    InfoBox({
      title: t('确认移除部署？'),
      content: () =>
        h(removeInfoxBox, {
          t,
          displayName: curEnvItem.value?.displayName,
          curEnv: curEnv.value,
          laneName: laneList.value.length ? curLaneName.value : undefined,
        }),
      confirmText: t('移除'),
      cancelText: t('取消'),
      confirmButtonTheme: 'danger',
      onConfirm: async () => {
        const params = {
          appID: appDetailStore.appID,
          envName: curEnvItem.value?.name,
          deployID: deployHistoryList.value?.[0]?.id || '',
          trafficLaneName: curLaneName.value,
        } as DeleteHelmDeployRequest;
        try {
          await DeployService.deleteHelmDeploy(params);
          getDeployHistories();
          Message({
            theme: 'success',
            message: t('移除部署成功'),
          });
        } catch (_error) {
          Message({
            theme: 'error',
            message: t('移除部署失败'),
          });
        }
      },
    });
  }
  // 显示部署弹窗
  function handleShowDeployDialog(type: 'Recreate' | 'RollingUpdate') {
    isShowDeploy.value = true;
    curDeployType.value = type;
  }

  // 环境变化，获取泳道列表和部署历史
  watch(
    () => curEnv.value,
    async (newEnv, oldEnv) => {
      if (newEnv && newEnv !== oldEnv) {
        curLaneName.value = '';
        // 重新获取泳道列表
        await handleGetLanesList(newEnv);
        curLaneName.value = laneList.value[0]?.name || '';
        deployHistoryList.value = [];
        getDeployHistories();
      }
    },
  );

  // 泳道变化时获取部署历史
  watch(
    () => curLaneName.value,
    newLane => {
      if (newLane && curEnv.value) {
        getDeployHistories();
      }
    },
  );

  // 切换tab到history时获取部署历史
  // watch(activeTab, () => {
  //   if (curEnv.value) {
  //     getDeployHistories();
  //   }
  // });

  // 获取 Chart 列表
  watch(
    () => appDetailStore.appID,
    newAppID => {
      if (newAppID && isHelmLikeAppType(appDetailStore.appType)) {
        handleGetChartList(newAppID);
        getDeployHistories();
      }
    },
    { immediate: true },
  );

  watch(
    () => appDetailStore.app,
    async () => {
      await appDetailStore.fetchAppDetail();
    },
    { immediate: true },
  );

  watch(
    targetEnvName,
    envName => {
      if (envName && envName !== curEnv.value && !isInitializingEnvFromUrl.value) {
        isInitializingEnvFromUrl.value = true;
        curEnv.value = envName;
        nextTick(() => {
          isInitializingEnvFromUrl.value = false;
        });
      }
    },
    { immediate: true },
  );

  watch(
    () => curEnv.value,
    envName => {
      if (envName && envName !== targetEnvName.value && !isInitializingEnvFromUrl.value) {
        targetEnvName.value = envName;
      }
    },
    { immediate: true },
  );
</script>

<style lang="postcss">
  .more-operations {
    padding: 6px 0 !important;
  }
</style>

<style lang="postcss" scoped>
  :deep(.bk-tab-header) {
    background-color: #fff;
    padding: 0 24px;
  }
  :deep(.bk-tab-header-nav .bk-tab-header-item) {
    padding: 0 !important;
    margin-right: 32px !important;
    font-size: 14px;
  }
  :deep(.bk-tab-content) {
    background-color: #fff;
    height: 100%;
    overflow: auto;
    &:has(.custom-resource-topology) {
      padding: 0 !important;
    }
  }
</style>
