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
  <SlideDetail
    ref="slideDetailRef"
    class="bg-[#fff]"
  >
    <div class="h-[82px] pt-[16px] bg-[#f0f1f5] px-[24px]">
      <div class="flex items-center gap-[8px]">
        <div class="text-[14px] text-[#313238] font-bold">{{ data?.displayName }}</div>
        <Tag
          v-if="envTypeConfig"
          :class="envTypeTagClassMap[data?.type || '']"
          type="stroke"
          >{{ envTypeConfig.name || data?.type }}</Tag
        >
        <Dropdown
          ref="dropdownRef"
          placement="bottom-start"
          trigger="click"
        >
          <div
            class="w-[26px] h-[26px] border-[1px] border-[#C4C6CC] border-solid bg-[#FFF] rounded-[2px] flex justify-center items-center cursor-pointer"
          >
            <Ellipsis class="transform-rotate-90 text-[#4D4F56] text-[16px]" />
          </div>
          <template #content>
            <Dropdown.DropdownMenu>
              <Dropdown.DropdownItem @click="handleDeleteEnv">
                {{ $t('删除环境') }}
              </Dropdown.DropdownItem>
            </Dropdown.DropdownMenu>
          </template>
        </Dropdown>
      </div>
      <p class="mt-[6px] text-[12px] text-[#979BA5]">{{ data?.name }}</p>
    </div>
    <Tab
      :key="props?.data?.name"
      v-model:active="activeTabName"
      class="app-detail-tab"
      :label-height="42"
      type="card-tab"
      :validate-active="false"
      @change="handleChangeDetail"
    >
      <!-- <Tab.TabPanel name="overview" :label="$t('概览')">概览</Tab.TabPanel>
      <Tab.TabPanel name="applicationList" :label="$t('应用列表')">应用列表</Tab.TabPanel>
      <Tab.TabPanel name="laneSettings" :label="$t('泳道设置')">
        <div class="flex gap-[12px]">
          <Select>
            <Select.Option value="泳道A">{{ '泳道A' }}</Select.Option>
            <Select.Option value="泳道B">{{ '泳道B' }}</Select.Option>
          </Select>
          <Tab
            :label-height="42"
            :validate-active="false"
            type="card">
            <Tab.TabPanel name="details" :label="$t('详情')"></Tab.TabPanel>
            <Tab.TabPanel name="configuration" :label="$t('配置')"></Tab.TabPanel>
            <Tab.TabPanel name="observabilityData" :label="$t('观测数据')"></Tab.TabPanel>
            <Tab.TabPanel name="deploymentRecords" :label="$t('部署记录')"></Tab.TabPanel>
          </Tab>
        </div>
      </Tab.TabPanel>
      <Tab.TabPanel name="trafficManagement" :label="$t('流量治理')">流量治理</Tab.TabPanel>
      <Tab.TabPanel name="canaryStrategy" :label="$t('灰度策略')">灰度策略</Tab.TabPanel>
      <Tab.TabPanel name="setting" :label="$t('设置')">设置</Tab.TabPanel> -->
      <Tab.TabPanel
        :label="$t('基本信息')"
        name="basicInfo"
        render-directive="if"
      >
        <BasicInfo
          :key="tabKey"
          :env="data.id ?? ''"
          :workspace="workspace"
          @update="handleUpdate"
        />
      </Tab.TabPanel>
      <Tab.TabPanel
        :label="$t('环境配置')"
        name="setting"
        render-directive="if"
      >
        <Setting
          :key="tabKey"
          :env="data.id ?? ''"
          :env-name="data.name ?? ''"
          :workspace="workspace"
          @update="handleUpdate"
        />
      </Tab.TabPanel>
      <!-- <Tab.TabPanel
        :label="$t('泳道配置')"
        name="laneConfig"
        render-directive="if"
      >
        <LaneConfig :data="data" />
      </Tab.TabPanel> -->
      <Tab.TabPanel
        :label="$t('观测数据')"
        name="observability"
        render-directive="if"
      >
        <div class="flex flex-col gap-[12px] h-full">
          <ApmInstance
            :current-apm="currentApm"
            :data="data"
            @update:current-apm="getEnvApm"
          />
          <MonitorIframe
            v-if="currentApm"
            :base-query-params="{
              apm_submenu: 0,
              needMenu: false,
            }"
            class="flex-1 min-h-0"
            :observability-query="{
              'filter-app_name': apmAppName ?? '',
              method: 'AVG',
              interval: 'auto',
              dashboardId: 'overview',
              from: 'now-1h',
              to: 'now',
              timezone: 'Asia/Shanghai',
              refreshInterval: -1,
              sceneType: 'overview',
              queryString: '',
              preciseFilter: false,
              isGroupByLimit: false,
            }"
            type="application"
          />
        </div>
      </Tab.TabPanel>
    </Tab>
  </SlideDetail>
</template>
<script lang="ts" setup>
  import { computed, onMounted, ref, watch } from 'vue';

  import { Dropdown, Tab, Tag } from 'bkui-vue';
  import { Ellipsis } from 'bkui-vue/lib/icon';
  import { useRoute, useRouter } from 'vue-router';
  import { GetEnvApmOutput } from '~/@types/v1/bkintegrations-bkmonitor';
  import { EnvOutput } from '~/@types/v1/env';
  import { BkintegrationsBkmonitorService } from '~/api/modules/v1';
  import MonitorIframe from '~/components/monitor-iframe.vue';
  import SlideDetail from '~/components/slide-detail.vue';
  import { envTypeMap, envTypeTagClassMap } from '~/composables/use-env-manager';

  import ApmInstance from './apm-instance.vue';
  import BasicInfo from './basic-info.vue';
  import Setting from './setting.vue';

  // 定义所有可用的 Tab 值
  const ACTIVE_TABS = [
    'applicationList',
    'basicInfo',
    'canaryStrategy',
    'laneConfig',
    'laneSettings',
    'modules',
    'observability',
    'overview',
    'setting',
    'trafficManagement',
  ] as const;

  // 从常量数组推导类型
  type ActiveTabType = (typeof ACTIVE_TABS)[number];

  interface IProps {
    activeTab?: ActiveTabType;
    data: EnvOutput;
  }
  const props = defineProps<IProps>();
  const emits = defineEmits(['update', 'delete']);

  const router = useRouter();
  const route = useRoute();
  const slideDetailRef = ref<InstanceType<typeof SlideDetail>>();

  const activeTabName = ref<ActiveTabType>(props.activeTab || 'basicInfo');

  const tabKey = computed(() => props.data?.createdAt?.toString() || Date.now().toString());

  const workspace = computed<string>(() => route.params.space as string);

  const envTypeConfig = computed(() => {
    if (props.data?.type && envTypeMap[props.data?.type]) {
      return envTypeMap[props.data.type];
    }
    return {} as { name: string; theme: string };
  });

  const currentApm = ref<GetEnvApmOutput | null>(null);
  const dropdownRef = ref<InstanceType<typeof Dropdown>>();

  // 获取当前环境关联的 APM 实例
  async function getEnvApm() {
    if (activeTabName.value !== 'observability') return;

    const envID = props.data.id;
    if (!envID) {
      currentApm.value = null;
      return;
    }

    currentApm.value = null;
    const apm = await BkintegrationsBkmonitorService.getEnvApm({ envID }, { interceptorErr: false }).catch(() => null);
    if (props.data.id === envID) {
      currentApm.value = apm;
    }
  }

  // 优先当前环境绑定的 APM name，没有则使用环境名
  const apmAppName = computed(() => {
    return currentApm.value?.name || props.data.name;
  });

  // 切换详情
  function handleChangeDetail(active: ActiveTabType) {
    activeTabName.value = active;
    // 更新query参数
    router.replace({
      query: {
        active: props?.data?.name,
        activeTab: activeTabName.value,
      },
    });
  }

  // 删除环境
  function handleDeleteEnv() {
    emits('delete', props.data);
    dropdownRef.value?.popoverRef?.hide();
  }

  function handleUpdate(row: EnvOutput) {
    emits('update', row);
  }

  // 显示面板
  function show() {
    slideDetailRef.value?.show();
  }

  onMounted(() => {
    // 判断 route.query.activeTab 的值是否为有效的 Tab 类型
    const queryTab = route.query.activeTab as string;
    if (queryTab && ACTIVE_TABS.includes(queryTab as ActiveTabType)) {
      activeTabName.value = queryTab as ActiveTabType;
    } else {
      activeTabName.value = props.activeTab || 'basicInfo';
      router.replace({
        query: {
          ...route.query,
          activeTab: activeTabName.value,
        },
      });
    }
  });

  watch([activeTabName, () => props.data?.id], getEnvApm, { immediate: true });

  defineExpose({
    show,
  });
</script>
<style lang="postcss" scoped>
  :deep(.app-detail-tab) {
    font-size: 14px;
    height: calc(100% - 82px);
    .bk-tab-content {
      display: flex;
      flex-direction: column;
      flex-basis: calc(100% - 42px);
      overflow-y: auto;
    }
    .bk-tab-content > .bk-tab-panel {
      flex: 1;
      min-height: 0;
    }
  }
</style>
