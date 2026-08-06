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
      <div class="flex justify-between mb-[16px]">
        <Layout.shape :width="80" />
        <Layout.shape :width="520" />
      </div>
      <!-- 服务列表骨架屏 -->
      <div class="rounded-[2px] overflow-hidden bg-[#FFF] mb-[16px]">
        <div class="service-header px-[36px] pt-[12px] pb-[16px] flex items-center gap-[12px]">
          <Layout.shape :width="180" />
          <Layout.shape :width="60" />
          <Layout.shape :width="100" />
        </div>
        <div class="service-content bg-[#FFF] px-[36px] pb-[16px]">
          <div class="mb-[24px]">
            <div class="flex items-center gap-[16px]">
              <Layout.shape :width="200" />
              <Layout.shape :width="200" />
              <Layout.shape :width="200" />
            </div>
          </div>
          <div class="max-w-[1000px]">
            <Layout.shape
              class="mb-[8px]"
              height="40px"
              width="100%"
            />
            <Layout.shape
              v-for="j in 3"
              :key="j"
              class="mb-[8px]"
              height="20px"
              width="100%"
            />
          </div>
        </div>
      </div>
    </template>
    <div>
      <Alert
        class="mb-[16px]"
        theme="info"
        :title="
          $t(
            '如果应用需要使用泳道做全链路灰度发布，则必须添加一个 Service，并设置为泳道服务（目前创建的 Service 默认为泳道服务）',
          )
        "
      />
      <div class="flex justify-between gap-[16px] mb-[16px]">
        <Button
          v-bk-tooltips="{
            content: $t('目前一个应用只能创建一个 Service'),
            disabled: !(filteredServiceList.length >= 1),
          }"
          :disabled="filteredServiceList.length >= 1"
          theme="primary"
          @click="handleAddService"
          >{{ $t('新建') }}</Button
        >
        <!-- 第一期默认为一个 service，所以暂时隐藏搜索 -->
        <!-- <Input
          v-model="searchValue"
          class="w-[520px]"
          clearable
          :placeholder="$t('搜索服务关键字')"
          type="search"
        ></Input> -->
      </div>
      <Exception
        v-if="filteredServiceList.length === 0"
        class="large-exception"
        :description="$t('暂无服务')"
        scene="part"
        type="empty"
      >
      </Exception>
      <div
        v-else
        class="flex flex-col gap-[16px]"
      >
        <!-- 服务列表 -->
        <ServiceItem
          v-for="service in filteredServiceList"
          :key="service.name"
          :service="service"
          @delete="handleDeleteService"
          @edit="handleEditService"
        />
      </div>

      <!-- 添加、编辑服务侧栏 -->
      <ServiceFormSideslider
        v-model:visible="sidesliderVisible"
        :current-service="currentService"
        :is-edit="isEdit"
        :loading="submitLoading"
        @cancel="handleCancel"
        @submit="handleSubmit"
      />
    </div>
  </Skeleton>
</template>
<script setup lang="ts">
  import { ref, watch } from 'vue';

  import { Alert, Button, Exception, Message } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { AppServiceOutput, CreateAppServiceRequest, UpdateAppServiceRequest } from '~/@types/v1/app-networking';
  import { AppNetworkingService } from '~/api/modules/v1';
  import Layout from '~/components/skeleton/skeleton-layout';
  import { useTableSearchInput } from '~/composables/use-search';
  import { useAppDetail } from '~/stores/app-detail';

  import ServiceFormSideslider from './service-form-sideslider.vue';
  import ServiceItem from './service-item.vue';

  import type { PortConfig } from './port-config-table.vue';

  interface ServiceFormData {
    name: string;
    ports: PortConfig[];
    selector: Array<{ key: string; value: string }>;
    trafficLaneEnabled: boolean;
  }

  const appDetailStore = useAppDetail();
  const { t } = useI18n();

  const isLoading = ref(false);
  const sidesliderVisible = ref(false);
  const isEdit = ref(false);
  const currentService = ref<null | ServiceFormData>(null);
  const submitLoading = ref(false);

  const serviceList = ref<AppServiceOutput[]>([]);

  // 搜索功能
  const searchKeys = ref([{ id: 'name', field: 'name', fuzzy: true }]);
  const { tableDataMatchSearch: filteredServiceList } = useTableSearchInput(serviceList, searchKeys, {
    ignoreCase: true, // 忽略大小写
  });

  // 获取服务列表
  async function getServiceList() {
    if (!appDetailStore.appID) return;
    isLoading.value = true;
    serviceList.value = await AppNetworkingService.listAppServices({
      appID: appDetailStore.appID,
    }).catch(() => []);
    isLoading.value = false;
  }

  // 添加服务侧栏
  function handleAddService() {
    isEdit.value = false;
    currentService.value = null;
    sidesliderVisible.value = true;
  }

  // 取消
  function handleCancel() {
    sidesliderVisible.value = false;
  }

  // 删除服务
  async function handleDeleteService(serviceName: string) {
    const result = await AppNetworkingService.deleteAppService({
      appID: appDetailStore.appID,
      name: serviceName,
    })
      .then(() => true)
      .catch(() => false);

    if (result) {
      // 刷新服务列表
      await getServiceList();
      Message({
        theme: 'success',
        message: t('删除成功'),
      });
    }
  }

  // 编辑服务
  function handleEditService(service: AppServiceOutput) {
    isEdit.value = true;
    // 将 API 返回的数据转换为表单数据格式
    currentService.value = {
      name: service?.name || '',
      selector: Object.entries(service.selector || {}).map(([key, value]) => ({ key, value })),
      ports: (service?.ports as PortConfig[]) ?? [],
      trafficLaneEnabled: service?.trafficLaneEnabled ?? false,
    };
    sidesliderVisible.value = true;
  }

  // 提交服务数据
  async function handleSubmit(data: ServiceFormData) {
    submitLoading.value = true;

    const requestData = transformFormDataToRequest(data);

    // 根据 isEdit 判断是创建/更新
    const apiCall = isEdit.value
      ? AppNetworkingService.updateAppService(requestData)
      : AppNetworkingService.createAppService(requestData);

    const result = await apiCall.then(() => true).catch(() => false);

    if (result) {
      await getServiceList();
      sidesliderVisible.value = false;
      Message({
        theme: 'success',
        message: isEdit.value ? t('修改成功') : t('新增成功'),
      });
    }
    submitLoading.value = false;
  }

  // 将表单数据转换为 API 请求参数
  function transformFormDataToRequest(data: ServiceFormData): CreateAppServiceRequest | UpdateAppServiceRequest {
    // 将 selector 数组转换为 Record<string, string>
    const selector: Record<string, string> = {};
    data.selector.forEach(item => {
      if (item.key && item.value) {
        selector[item.key] = item.value;
      }
    });

    // 转换端口配置，确保类型正确
    const ports = data.ports.map(port => ({
      name: port.name,
      port: Number(port.port),
      protocol: port.protocol,
      targetPort: String(port.targetPort),
    }));

    // 构建请求参数
    return {
      appID: appDetailStore.appID,
      name: data.name,
      selector,
      ports,
      trafficLaneEnabled: data.trafficLaneEnabled,
    };
  }

  watch(
    () => appDetailStore.appID,
    () => {
      getServiceList();
    },
    { immediate: true },
  );
</script>
