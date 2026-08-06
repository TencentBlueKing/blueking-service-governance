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
  <Skeleton :loading="isLoading">
    <template #loading>
      <Layout.shape
        :height="34"
        width="100%"
      />
      <FlexRow class="w-full my-[16px]">
        <template #left>
          <Layout.shape />
        </template>
        <template #right>
          <Layout.shape :width="360" />
        </template>
      </FlexRow>
      <Layout.table />
    </template>
    <div class="bg-[#fff]">
      <!-- 泳道配置 -->
      <template v-if="laneConfigTableData.length || searchVal !== undefined">
        <Alert
          closable
          theme="warning"
          :title="t('泳道使用须知：泳道规则对 HTTP 协议生效，需同时满足网格入口流量与服务之间调用走 HTTP 协议')"
        />
        <div class="my-[16px] flex justify-between">
          <Button
            theme="primary"
            @click="handleCreate"
            >{{ t('新建泳道') }}</Button
          >
          <Input
            v-model.trim="searchVal"
            class="w-[360px]"
            :placeholder="createPlaceholder({ labels: ['env.lane.label.laneName'] })"
            type="search"
            @change="handleSearch"
          >
          </Input>
        </div>
        <Table
          :data="laneConfigTableData"
          :pagination="pagination"
          :row-config="{
            isHover: true,
            isCurrent: true,
          }"
          :row-height="120"
        >
          <template #empty>
            <TableException
              :type="curExceptionType"
              @clear="handleClearSearch"
              @refresh="getListTrafficLane"
            >
            </TableException>
          </template>
          <TableColumn
            field="laneName"
            fixed="left"
            :label="t('泳道名称')"
            min-width="200"
          >
            <template #default="{ row }">
              <div class="flex items-center">
                <span class="mr-[7px]">{{ row.laneName }}</span>
                <Tag v-if="row.laneType === 'base'">{{ t('基线泳道') }}</Tag>
              </div>
            </template>
          </TableColumn>
          <TableColumn
            field="laneServices"
            :label="t('泳道应用')"
            min-width="200"
          >
            <template #default="{ row }">
              <div class="flex flex-col">
                <div
                  v-for="(item, index) in row.displayedItems"
                  :key="item"
                  class="h-[22px] flex items-center"
                >
                  <span>{{ item }}</span>
                  <div
                    v-if="index === 4 && row.laneServices.length > 5"
                    class="flex items-center"
                  >
                    <span class="mr-[2px]">...</span>
                    <Popover width="200">
                      <Tag class="cursor-pointer">
                        {{ t('共 {0} 个', [row.laneServices.length]) }}
                      </Tag>
                      <template #content>
                        <div
                          v-for="laneService in row.laneServices"
                          :key="laneService"
                          class="h-[22px] flex items-center"
                        >
                          {{ laneService }}
                        </div>
                      </template>
                    </Popover>
                  </div>
                </div>
              </div>
            </template>
          </TableColumn>
          <TableColumn
            field="laneServiceVersionLabels"
            :label="t('泳道版本')"
            min-width="300"
          >
            <template #default="{ row }">
              <Tag>{{ row.labelKey }}: {{ row.labelValue }}</Tag>
            </template>
          </TableColumn>
          <TableColumn
            field="headersValue"
            :label="t('流量染色 Headers')"
            min-width="220"
          >
            <template #default="{ row }">
              <Tag v-if="row.headersKey">{{ row.headersKey }}: {{ row.headersValue }}</Tag>
            </template>
          </TableColumn>
          <TableColumn
            field="laneDesc"
            :label="t('泳道描述')"
            min-width="200"
          >
          </TableColumn>
          <TableColumn
            field="isOpen"
            :label="t('启用')"
            min-width="100"
          >
            <template #default="{ row }">
              <PopConfirm
                :confirm-text="row.isOpen ? t('停用') : t('启用')"
                placement="top"
                :title="row.isOpen ? t('确认停用该泳道？') : t('确认启用该泳道？')"
                trigger="click"
                :width="280"
                @cancel="handleCancelChangeStatus"
                @confirm="handleChangeStatus(row)"
              >
                <template #content>
                  <div class="mb-[4px]">
                    <span class="text-[#4D4F56]">{{ t('泳道名称') }}：</span>
                    <span class="text-[#313238]">{{ row.laneName }}</span>
                  </div>
                  <div
                    v-if="row.isOpen"
                    class="mb-[20px] text-[#4D4F56]"
                  >
                    {{ t('停用后该泳道的流量规则将不生效') }}
                  </div>
                  <div
                    v-else
                    class="mb-[20px] text-[#4D4F56]"
                  >
                    {{ t('启用后该泳道将开始生效') }}
                  </div>
                </template>
                <Switcher
                  v-bk-tooltips="{
                    content: $t('存在特性泳道的情况下，不支持停用基线泳道'),
                    placement: 'top',
                    disabled: !(row.laneType === 'base' && isHasFeatureLane && row.isOpen),
                    delay: 300,
                  }"
                  :before-change="requestHandler"
                  :disabled="row.laneType === 'base' && isHasFeatureLane && row.isOpen"
                  theme="primary"
                  :value="row.isOpen"
                />
              </PopConfirm>
            </template>
          </TableColumn>
          <TableColumn
            fixed="right"
            :label="t('操作')"
            min-width="200"
          >
            <template #default="{ row }">
              <div class="flex gap-[10px]">
                <Button
                  text
                  theme="primary"
                  @click.stop="handleEdit(row)"
                >
                  {{ t('编辑') }}
                </Button>
                <PopConfirm
                  :confirm-text="t('删除')"
                  placement="top"
                  :title="t('确认删除该泳道？')"
                  trigger="click"
                  :width="280"
                  @confirm="handleDelete(row)"
                >
                  <template #content>
                    <div class="mb-[4px]">
                      <span class="text-[#4D4F56]">{{ t('泳道名称') }}：</span>
                      <span class="text-[#313238]">{{ row.laneName }}</span>
                    </div>
                    <div class="mb-[20px] text-[#4D4F56]">{{ t('删除后该泳道将不可恢复') }}</div>
                  </template>
                  <Button
                    v-bk-tooltips="{
                      content: $t('存在特性泳道的情况下，不支持删除基线泳道'),
                      placement: 'top',
                      disabled: !(row.laneType === 'base' && isHasFeatureLane),
                      delay: 300,
                    }"
                    :disabled="row.serviceStatus === 'enable' || (row.laneType === 'base' && isHasFeatureLane)"
                    text
                    theme="primary"
                    @click.stop
                  >
                    {{ t('删除') }}
                  </Button>
                </PopConfirm>
              </div>
            </template>
          </TableColumn>
        </Table>
      </template>
      <template v-else>
        <Exception
          class="large-exception"
          scene="part"
          type="empty"
        >
          <template #type>
            <img src="/empty.svg" />
          </template>
          <template #description>
            <div class="text-[20px] text-[#313238] mb-[16px]">{{ $t('暂无泳道') }}</div>
            <div class="text-[14px] text-[#4D4F56] mb-[16px]">{{ $t('泳道未启用，无数据') }}</div>
            <Button
              theme="primary"
              @click="handleOpen"
              >{{ t('启用泳道') }}</Button
            >
          </template>
        </Exception>
      </template>
      <Sideslider
        v-model:is-show="sidesliderData.isShow"
        :before-close="handleBeforeClose"
        quick-close
        render-directive="if"
        :title="sidesliderData.title"
        :width="640"
        @closed="handleClose"
      >
        <template #header>
          <div class="flex items-center">
            <span class="mr-[7px]">
              {{ sidesliderData.title }}
            </span>
            <Tag v-if="sidesliderData.type === 'open' || sidesliderData.data?.laneType === 'base'">{{
              t('基线泳道')
            }}</Tag>
            <Tag v-else>{{ $t('特性泳道') }}</Tag>
          </div>
        </template>
        <template #default>
          <div class="px-[24px] pt-[18px] pb-[8px]">
            <Alert
              v-if="sidesliderData.type === 'open'"
              class="mb-[16px]"
              closable
              theme="warning"
              :title="t('一旦启用泳道，现有应用会被包含到基线泳道内，请谨慎操作')"
            />
            <Form
              ref="formRef"
              form-type="vertical"
              :model="formModel"
              :rules="rules"
            >
              <Form.FormItem
                :label="t('泳道名称')"
                property="name"
                required
              >
                <Popover
                  placement="bottom-start"
                  theme="light"
                  trigger="click"
                >
                  <Input
                    v-model.trim="formModel.name"
                    clearable
                    :placeholder="t('请输入泳道名称')"
                  >
                  </Input>
                  <template #content>
                    <ul class="text-[#4D4F56] leading-[22px]">
                      <li>{{ `${$t('正则')}：^[a-zA-Z0-9]([-_.a-zA-Z0-9]{0,61}[a-zA-Z0-9])?$` }}</li>
                      <li>{{ `${$t('长度限制')}：${$t('最多 {0} 个字符', [63])}` }}</li>
                      <li>{{ `${$t('首尾字符')}：${$t('必须以字母或数字（[a-zA-Z0-9]）开头和结尾')}` }}</li>
                      <li>
                        {{ `${$t('中间字符')}：${$t('允许包含破折号（-）、下划线（_）、点（.）以及字母数字')}` }}
                      </li>
                    </ul>
                  </template>
                </Popover>
              </Form.FormItem>
              <div class="relative">
                <span
                  v-if="sidesliderData.type === 'open'"
                  class="absolute top-0 left-[62px] text-[#979BA5] text-[12px]"
                >
                  {{ $t('（建议基线泳道应当包含全部应用，后续特性泳道将基于该范围创建）') }}
                </span>
                <Form.FormItem
                  :label="t('泳道应用')"
                  property="mode"
                  required
                >
                  <Radio.Group
                    v-if="sidesliderData.type === 'open'"
                    v-model="formModel.mode"
                  >
                    <Radio
                      :disabled="true"
                      label="all"
                    >
                      {{ $t('全部应用（包括后续动态变化部分）') }}
                    </Radio>
                    <Radio
                      checked
                      label="custom"
                      >{{ $t('自定义服务') }}</Radio
                    >
                  </Radio.Group>
                  <Transfer
                    v-bkloading="{ loading: servicesLoading }"
                    display-key="appName"
                    searchable
                    setting-key="appName_serviceName"
                    show-overflow-tips
                    sort-key="appName"
                    :source-list="sourceList"
                    :target-list="targetValue"
                    :title="[t('可选应用'), t('已选应用')]"
                    @change="handleChangeService"
                    @update:target-list="handleTargetListChange"
                  >
                    <template #left-header>
                      <span class="text-[#4D4F56] text-[12px] leading-[40px] font-bold">{{
                        `${t('可选应用')}（${sourceList.length}）`
                      }}</span>
                    </template>
                    <template #right-header>
                      <span class="text-[#4D4F56] text-[12px] leading-[40px] font-bold">{{
                        `${t('已选应用')}（${targetValue.length}）`
                      }}</span>
                    </template>
                    <template #source-option="sourceData">
                      <div class="w-full">
                        <overflow-title
                          v-if="!sourceData.disabled"
                          v-bk-tooltips="{
                            content: `${sourceData.appName}（Service: ${sourceData.services?.[0]?.name}）`,
                            placement: 'right',
                            delay: 300,
                          }"
                        >
                          <span class="text-[12px]">{{
                            `${sourceData.appName}（Service: ${sourceData.services?.[0]?.name}）`
                          }}</span>
                        </overflow-title>
                        <Popover
                          v-else
                          placement="right"
                          :popover-delay="[100, 0]"
                        >
                          <div class="w-full">
                            <Button
                              disabled
                              text
                            >
                              <span class="text-[12px]">{{ `${sourceData.appName}（${$t('暂无服务')}）` }}</span>
                            </Button>
                          </div>
                          <template #content>
                            <span class="text-[12px] mr-[8px]">{{ $t('应用没有关联的 Service') }}</span>
                            <Button
                              text
                              theme="primary"
                              @click="handleToNetWork(sourceData.appName)"
                            >
                              {{ $t('去添加') }}
                              <Share class="text-[#3A84FF] ml-[5px] cursor-pointer" />
                            </Button>
                          </template>
                        </Popover>
                      </div>
                    </template>
                    <template #target-option="targetData">
                      <div class="w-full">
                        <overflow-title
                          v-bk-tooltips="{
                            content: `${targetData.appName}（Service: ${targetData.services?.[0]?.name}）`,
                            placement: 'left',
                            delay: 300,
                          }"
                        >
                          <span class="text-[#4D4F56] text-[12px]">{{
                            `${targetData.appName}（Service: ${targetData.services?.[0]?.name}）`
                          }}</span>
                        </overflow-title>
                      </div>
                    </template>
                  </Transfer>
                </Form.FormItem>
              </div>
              <Form.FormItem
                :label="t('泳道描述')"
                property="desc"
                required
              >
                <Input
                  v-model="formModel.desc"
                  :maxlength="100"
                  :placeholder="t('请输入泳道描述')"
                  type="textarea"
                ></Input>
              </Form.FormItem>
              <Form.FormItem
                :label="t('泳道版本 Label')"
                property="laneServiceLabels"
                required
              >
                <div class="flex gap-[32px]">
                  <Input
                    v-model.trim="formModel.laneServiceLabels.key"
                    :disabled="sidesliderData.type === 'edit'"
                    :maxlength="100"
                    :placeholder="t('请输入 key')"
                    prefix="key"
                  ></Input>
                  <Input
                    v-model.trim="formModel.laneServiceLabels.value"
                    :disabled="sidesliderData.type === 'edit'"
                    :maxlength="100"
                    :placeholder="t('请输入 value')"
                    prefix="value"
                  ></Input>
                </div>
              </Form.FormItem>
              <Form.FormItem
                v-if="sidesliderData.type !== 'open' && sidesliderData.data?.laneType !== 'base'"
                :label="t('流量染色 Headers')"
                property="serviceConfig"
                required
              >
                <div class="flex gap[32px]">
                  <Input
                    v-model.trim="formModel.serviceConfig.headers.values.key"
                    :disabled="sidesliderData.type === 'edit'"
                    :maxlength="100"
                    :placeholder="t('请输入 key')"
                    prefix="key"
                  ></Input>
                  <Input
                    v-model.trim="formModel.serviceConfig.headers.values.value"
                    :disabled="sidesliderData.type === 'edit'"
                    :maxlength="100"
                    :placeholder="t('请输入 value')"
                    prefix="value"
                  ></Input>
                </div>
              </Form.FormItem>
            </Form>
          </div>
        </template>
        <template #footer>
          <Button
            :loading="confirmLoading"
            theme="primary"
            @click="handleConfirm"
            >{{ t('确定') }}</Button
          >
          <Button
            class="ml-[8px]"
            @click="handleClose"
            >{{ t('取消') }}</Button
          >
        </template>
      </Sideslider>
    </div>
  </Skeleton>
</template>
<script lang="ts" setup>
  import { computed, reactive, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import {
    Alert,
    Button,
    Exception,
    Form,
    Input,
    OverflowTitle,
    PopConfirm,
    Popover,
    Radio,
    Sideslider,
    Switcher,
    Tag,
    Transfer,
  } from 'bkui-vue';
  import { Share } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { useRouter } from 'vue-router';
  import { TrafficLaneCandidateAppOutput } from '~/@types/v1/app-networking';
  import { EnvOutput } from '~/@types/v1/env';
  import { TrafficManagerService } from '~/api/modules/trafficmanager';
  import { AppNetworkingService } from '~/api/modules/v1';
  import { BKMS_REGEX } from '~/common/const';
  import Layout from '~/components/skeleton/skeleton-layout';
  import useLeaveConfirm from '~/composables/use-leave-confirm';
  import { useSearchPlaceholder } from '~/composables/use-search-placeholder';
  import useTableEmpty from '~/composables/use-table-empty';
  import { useSpaceStore } from '~/stores/space';
  import { useUserStore } from '~/stores/user';

  import type {
    CreateTrafficLaneRequest,
    ListTrafficLaneData,
    ListTrafficLaneRequest,
    TrafficLane,
    TrafficLaneData,
    TrafficLaneService,
  } from '~/@types/trafficmanager';

  interface ILaneData extends TrafficLane {
    displayedItems: string[];
    headersKey: string;
    headersValue: string;
    isOpen: boolean;
    labelKey: string;
    labelValue: string;
    laneServices: string[];
    services: TrafficLaneService[];
  }

  const props = defineProps<{
    data: EnvOutput;
  }>();
  const { t } = useI18n();
  const { createPlaceholder } = useSearchPlaceholder();
  const userStore = useUserStore();
  const spaceStore = useSpaceStore();
  const router = useRouter();

  const searchVal = ref();
  const { setTypeToError, clearErrorType, curExceptionType } = useTableEmpty({
    filters: searchVal,
  });
  const servicesLoading = ref(false);
  // 环境配置列表
  const laneConfigTableData = ref<ILaneData[]>([]);
  // 分页数据
  const pagination = ref({ count: 0, limit: 20, current: 1 });
  const targetValue = ref<string[]>([]);
  const targetListValue = ref<TrafficLaneCandidateAppOutput[]>([]);
  const sourceList = ref<TrafficLaneCandidateAppOutput[]>([]);
  const sidesliderData = reactive<{
    data: null | Record<string, unknown>;
    isShow: boolean;
    title: string;
    type: 'create' | 'edit' | 'open';
  }>({
    isShow: false,
    title: t('启用泳道'),
    type: 'open', // open: 开启，create: 创建，edit：编辑
    data: null,
  });
  const formRef = ref<InstanceType<typeof Form>>();
  const defaultFormValue = ref<CreateTrafficLaneRequest>({
    name: '',
    desc: '',
    laneType: '',
    laneSpace: '',
    laneEnv: '',
    laneApp: '',
    laneTenantId: '',
    laneServiceLabels: {},
    annotations: { values: {} },
    clusters: [],
    laneProvider: '',
    laneServiceProvider: '',
    creator: '',
    mode: 'custom',
    serviceConfig: {
      services: [],
      headers: { values: {} },
    },
  });
  const formModel = ref<CreateTrafficLaneRequest>({ ...defaultFormValue.value });
  const { confirmBox, withPausedWatch } = useLeaveConfirm(formModel);

  // 表单验证规则
  const rules = {
    name: [
      {
        validator: (value: string) => BKMS_REGEX.laneNameRegex.test(value),
        message: t('泳道名称格式不正确'),
        trigger: 'blur',
      },
    ],
  };

  const handleChangeService = (
    _sourceList: TrafficLaneCandidateAppOutput[],
    targetList: TrafficLaneCandidateAppOutput[],
    _targetValueList: string[],
  ) => {
    targetListValue.value = targetList;
  };

  /**
   * @description 泳道应用 - 自定义服务是否更新过 用于离开确认
   * formModel中 自定义服务并不作为其中一项直接双向绑定，因此需要特殊处理
   */
  let isTargetListChange = false;
  function handleTargetListChange(newTargetValue: string[]) {
    // 如果已经触发change，则无需再判断
    if (isTargetListChange) return;
    // 如果两者length长度不一致，则可以理解为Change，因为Transfer的更新会导致新旧值的长度不一致
    if (newTargetValue.length !== targetListValue.value.length) {
      isTargetListChange = true;
    }
  }

  const isLoading = ref(false);

  /** 当前环境的命名空间 */
  const curEnvNameSpace = computed(() => props.data.cluster?.namespace || '');

  /** 是否有特性泳道 */
  const isHasFeatureLane = computed(() => {
    return laneConfigTableData.value.some(item => item.laneType !== 'base');
  });

  const getListTrafficLane = async () => {
    try {
      isLoading.value = true;
      const params: Partial<ListTrafficLaneRequest> = {
        laneEnv: props.data.name,
        laneSpace: spaceStore.currentSpace,
        laneApp: props.data.name,
        page: pagination.value.current,
        limit: pagination.value.limit,
      };
      if (searchVal.value) {
        params['laneName'] = searchVal.value;
      }
      const list = await (TrafficManagerService.ListTrafficLane(params) as Promise<ListTrafficLaneData>).catch(() => ({
        count: 0,
        data: [],
      }));
      pagination.value.count = list.data.length;
      laneConfigTableData.value = list.data
        .map((item: TrafficLaneData) => {
          const labelKey = Object.keys(item.lane.laneServiceVersionLabels)[0];
          const labelValue = Object.values(item.lane.laneServiceVersionLabels)[0];
          const headersKey = item.lane.serviceRules?.headers ? Object.keys(item.lane.serviceRules.headers)[0] : '';
          const headersValue = item.lane.serviceRules?.headers ? Object.values(item.lane.serviceRules.headers)[0] : '';
          const laneServices = item.services.map((item: { serviceName: string }) => item.serviceName);
          const isOpen = item.lane.serviceStatus === 'enable';
          const displayedItems = laneServices.slice(0, 5);
          return {
            ...item.lane,
            labelKey,
            labelValue,
            headersKey,
            headersValue,
            laneServices,
            displayedItems,
            services: item.services,
            isOpen,
          };
        })
        .sort((a, b) => {
          if (a.laneType === 'base' && b.laneType !== 'base') {
            return -1;
          }
          if (a.laneType !== 'base' && b.laneType === 'base') {
            return 1;
          }
          return 0;
        });
      clearErrorType();
    } catch (error) {
      console.error(error);
      setTypeToError();
    } finally {
      isLoading.value = false;
    }
  };

  const handleOpen = async () => {
    sidesliderData.isShow = true;
    sidesliderData.title = t('启用泳道');
    sidesliderData.type = 'open';
    sidesliderData.data = null;
    targetValue.value = [];
    withPausedWatch(async () => {
      formModel.value = { ...defaultFormValue.value };
      await getLaneApplication();
    });
  };

  const handleCreate = async () => {
    sidesliderData.isShow = true;
    sidesliderData.title = t('新建泳道');
    sidesliderData.type = 'create';
    sidesliderData.data = null;
    targetValue.value = [];
    withPausedWatch(async () => {
      formModel.value = { ...defaultFormValue.value };
      await getLaneApplication();
    });
  };

  const handleSearch = async () => {
    await getListTrafficLane();
  };

  const handleClearSearch = () => {
    searchVal.value = '';
    getListTrafficLane();
  };

  let resolveFlag: ((value: boolean) => void) | null = null;
  let rejectFlag: ((reason?: unknown) => void) | null = null;
  const requestHandler = () => {
    return new Promise((resolve, reject) => {
      resolveFlag = resolve;
      rejectFlag = reject;
    });
  };
  const handleChangeStatus = async (row: ILaneData) => {
    await TrafficManagerService.UpdateTrafficLaneServicesStatus({
      laneId: row.laneId,
      enable: !row.isOpen,
    }).catch(err => rejectFlag?.(err));
    resolveFlag?.(true);
    await getListTrafficLane();
  };
  const handleCancelChangeStatus = () => {
    resolveFlag?.(false);
  };

  const handleEdit = async (row: ILaneData) => {
    await getLaneApplication();
    sidesliderData.isShow = true;
    sidesliderData.title = t('编辑泳道');
    sidesliderData.type = 'edit';
    sidesliderData.data = {
      ...row,
      name: row.laneName,
      desc: row.laneDesc,
      laneApp: row.laneEnv,
      laneServiceLabels: {
        key: row.labelKey,
        value: row.labelValue,
      },
      serviceConfig: {
        headers: {
          values:
            row.laneType !== 'base'
              ? {
                  key: row.headersKey,
                  value: row.headersValue,
                }
              : {},
        },
      },
    };
    withPausedWatch(async () => {
      Object.assign(formModel.value, sidesliderData.data);
      targetValue.value = sourceList.value.reduce<string[]>((acc, cur) => {
        if (row.services.findIndex(item => item.serviceName === cur.services?.[0]?.name) !== -1) {
          acc.push(`${cur.appName}_${cur.services?.[0]?.name}`);
        }
        return acc;
      }, []);
    });
  };

  const handleDelete = async (row: ILaneData) => {
    await TrafficManagerService.DeleteTrafficLane({
      laneId: row.laneId,
    }).catch(() => false);
    await getListTrafficLane();
  };

  /** 新开 tab 跳转到：应用-》网络访问-〉添加 Service  */
  const handleToNetWork = (name: string) => {
    const route = router.resolve({
      name: 'detail',
      params: {
        type: 'helm',
        name,
        menuName: 'network',
      },
      query: {
        activeTab: 'service',
      },
    });
    window.open(route.href, '_blank');
  };

  const getLaneApplication = async () => {
    servicesLoading.value = true;
    const res = await AppNetworkingService.listTrafficLaneCandidateApps({
      workspaceID: spaceStore.currentSpace,
    }).catch(() => ({ data: [] }));
    sourceList.value = (res as TrafficLaneCandidateAppOutput[]).map(item => ({
      ...item,
      appName_serviceName: `${item.appName}_${item.services?.[0]?.name || ''}`,
      disabled: !item.services?.[0]?.trafficLaneEnabled,
    }));
    servicesLoading.value = false;
  };

  const confirmLoading = ref(false);
  const handleConfirm = async () => {
    const validate = await formRef.value?.validate().catch(() => false);
    if (!validate) return;
    formModel.value.laneApp = props.data?.name ?? '';
    formModel.value.laneEnv = props.data?.name ?? '';
    formModel.value.laneSpace = spaceStore.currentSpace;
    let res;
    const user = userStore.userInfo.user_id;
    const services = targetListValue.value.map((item: TrafficLaneCandidateAppOutput) => {
      const curService = item.services?.[0].name;
      return {
        serviceHost: `${curService}.${curEnvNameSpace.value}`,
        serviceName: curService,
        serviceSpace: curEnvNameSpace.value,
        serviceKey: formModel.value.laneServiceLabels.key,
        serviceVersion: formModel.value.laneServiceLabels.value,
      };
    });
    const laneServiceLabels = { [formModel.value.laneServiceLabels.key]: formModel.value.laneServiceLabels.value };
    confirmLoading.value = true;
    if (sidesliderData.type !== 'edit') {
      const laneType = sidesliderData.type === 'open' ? 'base' : 'feature';
      const headersKey = formModel.value.serviceConfig.headers.values.key;
      const headersValue = formModel.value.serviceConfig.headers.values.value;
      let headers = null;
      if (laneType === 'feature') {
        headers = {
          values: {
            [headersKey]: headersValue,
          },
        };
      }

      res = await TrafficManagerService.CreateTrafficLane({
        ...formModel.value,
        clusters: [props.data.cluster?.clusterID ?? ''],
        laneProvider: 'istio',
        laneServiceProvider: 'istio',
        laneType,
        creator: user,
        laneServiceLabels,
        serviceConfig: {
          services,
          headers,
        },
      }).catch(() => false);
    } else {
      res = await TrafficManagerService.UpdateTrafficLane({
        laneId: sidesliderData.data?.laneId,
        laneName: formModel.value.name,
        laneDesc: formModel.value.desc,
        updater: user,
        services,
      }).catch(() => false);
    }
    confirmLoading.value = false;
    if (res !== false) {
      // 这里因为更新接口返回的接口定义有情况，res成功时候变成了undefined，暂时这样处理
      sidesliderData.isShow = false;
      await getListTrafficLane();
    }
  };

  // 侧边栏关闭前确认
  async function handleBeforeClose(): Promise<boolean> {
    const res = await confirmBox(true, {
      validates: [() => !isTargetListChange],
    });
    // 确认离开 重置isTargetListChange
    if (res) {
      isTargetListChange = false;
    }
    return res;
  }

  // 关闭弹窗
  async function handleClose() {
    if (await handleBeforeClose()) {
      sidesliderData.isShow = false;
    }
  }

  watch(
    () => props.data,
    async () => {
      // 获取泳道列表
      await getListTrafficLane();
    },
    { immediate: true },
  );
</script>
<style lang="postcss" scoped>
  .custom-shadow {
    box-shadow: 0 -1px 0 0 #dcdee5;
  }
  :deep(.bk-sideslider-footer) {
    margin-top: 0;
  }
</style>
