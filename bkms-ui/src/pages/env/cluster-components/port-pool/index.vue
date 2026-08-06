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
    back-color="#3A84FF"
    class="env-header !fixed w-full z-[999]"
    :title="$t('端口池配置')"
    :trigger-back="handlePageBack"
  >
    <Divider
      class="h-[16px] mx-[16px] text-[#DCDEE5]"
      direction="vertical"
      type="solid"
    />
    <span class="text-[#979BA5] text-[14px] mr-[16px]">{{ $t('环境') }}：{{ envDetail.displayName || envId }}</span>
    <span class="text-[#979BA5] text-[14px]">{{ $t('集群') }}：{{ envDetail.cluster?.clusterID || '--' }}</span>
  </MsHeader>
  <Skeleton
    class="bg-[#fff] mt-[52px] p-[16px]"
    :loading="isLoading"
  >
    <template #loading>
      <Layout.shape
        class="mb-[16px]"
        :height="32"
        :width="120"
      />
      <Layout.table :rows="10" />
    </template>
    <div class="mt-[52px] px-[24px] py-[16px] h-[calc(100%-52px)]">
      <div class="flex items-center justify-between mb-[16px]">
        <Button
          theme="primary"
          @click="handleCreatePortPool"
        >
          <Plus
            :height="24"
            :width="24"
          />
          {{ $t('新建端口池') }}
        </Button>
      </div>
      <Table
        :data="tableData"
        :pagination="pagination"
        :row-config="{ isHover: true, isCurrent: true }"
        show-header-overflow
        show-overflow
        @page-limit-change="handlePageLimitChange"
        @page-value-change="handlePageValueChange"
      >
        <template #empty>
          <Exception
            class="min-h-[180px]"
            :description="$t('暂无端口池数据')"
            scene="part"
            type="empty"
          />
        </template>
        <TableColumn
          field="name"
          :label="$t('端口池名称')"
          min-width="180"
          show-overflow="tooltip"
        />
        <TableColumn
          :label="$t('协议')"
          min-width="120"
          show-overflow="tooltip"
        >
          <template #default="{ row }: { row: PortPoolConfigOutputObj }">
            {{ row.poolItems?.[0]?.protocol || '--' }}
          </template>
        </TableColumn>
        <TableColumn
          :label="$t('端口段长度')"
          min-width="120"
          show-overflow="tooltip"
        >
          <template #default="{ row }: { row: PortPoolConfigOutputObj }">
            {{ row.poolItems?.[0]?.segmentLength ?? '--' }}
          </template>
        </TableColumn>
        <TableColumn
          :label="$t('起始端口')"
          min-width="120"
          show-overflow="tooltip"
        >
          <template #default="{ row }: { row: PortPoolConfigOutputObj }">
            {{ row.poolItems?.[0]?.startPort ?? '--' }}
          </template>
        </TableColumn>
        <TableColumn
          :label="$t('结束端口')"
          min-width="120"
          show-overflow="tooltip"
        >
          <template #default="{ row }: { row: PortPoolConfigOutputObj }">
            {{ row.poolItems?.[0]?.endPort ?? '--' }}
          </template>
        </TableColumn>
        <TableColumn
          field="status"
          :label="$t('状态')"
          min-width="80"
          show-overflow="tooltip"
        />
        <TableColumn
          :label="$t('负载均衡配置')"
          min-width="240"
          show-overflow="tooltip"
        >
          <template #default="{ row }: { row: PortPoolConfigOutputObj }">
            <MoreTag
              v-if="row.poolItems?.some(item => item.loadBalancerIDs?.length)"
              :data="row.poolItems.flatMap(item => item.loadBalancerIDs || [])"
              overflow-mode="tooltip"
            />
            <span v-else>--</span>
          </template>
        </TableColumn>
        <TableColumn
          :label="$t('操作')"
          min-width="140"
        >
          <template #default="{ row }: { row: PortPoolConfigOutputObj }">
            <Button
              class="mr-[16px]"
              :disabled="row.poolItems?.some((_item: PortPoolItemOutput) => row.status === 'Deleting')"
              text
              theme="primary"
              @click="handleEditPortPool(row)"
            >
              {{ $t('编辑') }}
            </Button>
            <Button
              :disabled="row.poolItems?.some((_item: PortPoolItemOutput) => row.status === 'Deleting')"
              text
              theme="primary"
              @click="handleDeletePortPool(row)"
            >
              {{ $t('删除') }}
            </Button>
          </template>
        </TableColumn>
      </Table>
    </div>
  </Skeleton>

  <ProtPoolSideslider
    v-model:visible="isShowSideslider"
    :edit-data="editData"
    :is-edit="isEdit"
    :loading="isSubmitting"
    @confirm="handleConfirmPortPool"
  />

  <Dialog
    v-model:is-show="isShowDelete"
    draggable
    :footer-align="'center'"
    :header-align="'center'"
    render-directive="if"
    theme="primary"
    :title="$t('确认删除端口池？')"
  >
    <div class="text-[12px]">
      <p>{{ $t('端口池名称') }}：{{ deleteTarget?.name }}</p>
      <div class="bg-[#F5F7FA] mt-[10px] py-[12px] px-[14px]">{{ $t('端口池删除后数据不可恢复，请确认') }}</div>
    </div>
    <template #footer>
      <Button
        class="mr-[10px]"
        :loading="isDeleting"
        theme="danger"
        @click="handleConfirmDelete"
      >
        {{ $t('删除') }}
      </Button>
      <Button
        :disabled="isDeleting"
        @click="isShowDelete = false"
      >
        {{ $t('取消') }}
      </Button>
    </template>
  </Dialog>
</template>

<script lang="ts" setup>
  import { computed, onMounted, ref } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Button, Dialog, Divider, Exception, Message } from 'bkui-vue';
  import { Plus } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { useRoute, useRouter } from 'vue-router';
  import { EnvDetailOutput } from '~/@types/v1/env';
  import { PortPoolConfigOutputObj, PortPoolItemInput, PortPoolItemOutput } from '~/@types/v1/port-pool';
  import { EnvService, PortPoolService } from '~/api/modules/v1';
  import Layout from '~/components/skeleton/skeleton-layout';
  import useInterval from '~/composables/use-interval';

  import ProtPoolSideslider from './prot-pool-sideslider.vue';

  interface ConfirmData {
    name: string;
    poolItems: PortPoolItemInput[];
  }

  const props = defineProps<{
    envId: string;
  }>();

  const router = useRouter();
  const route = useRoute();
  const { t } = useI18n();

  const isShowSideslider = ref(false);
  const isEdit = ref(false);
  const editData = ref<null | PortPoolConfigOutputObj>(null);

  const isSubmitting = ref(false);
  const isShowDelete = ref(false);
  const isDeleting = ref(false);
  const deleteTarget = ref<null | PortPoolConfigOutputObj>(null);

  const isLoading = ref(true);
  const envDetail = ref<EnvDetailOutput>({} as EnvDetailOutput);
  const portPoolList = ref<PortPoolConfigOutputObj[]>([]);
  const pagination = ref({
    current: 1,
    count: 0,
    limit: 10,
  });

  // 分页后的表格数据
  const tableData = computed<PortPoolConfigOutputObj[]>(() => {
    const { current, limit } = pagination.value;
    const start = (current - 1) * limit;
    return portPoolList.value.slice(start, start + limit);
  });

  async function fetchEnvDetail() {
    if (!props.envId) return;
    try {
      envDetail.value = await EnvService.getEnv({ envID: props.envId });
    } catch {
      envDetail.value = {} as EnvDetailOutput;
    }
  }

  async function fetchPortPools() {
    if (!props.envId) return;
    isLoading.value = true;
    try {
      const data = await PortPoolService.listPortPools({ envID: props.envId });
      portPoolList.value = data || [];
      pagination.value.count = portPoolList.value.length;
    } catch {
      portPoolList.value = [];
      pagination.value.count = 0;
    } finally {
      isLoading.value = false;
    }
  }

  /** 确认删除端口池 */
  // 由于后台接口原因（涉及到云上监听器的删除），删除资源池时，重新拉取列表不会立即生效
  async function handleConfirmDelete() {
    if (!deleteTarget.value || !props.envId) return;
    isDeleting.value = true;
    try {
      await PortPoolService.deletePortPool({
        envID: props.envId,
        name: deleteTarget.value?.name || '',
      });
      Message({
        message: t('删除成功'),
        theme: 'success',
      });
      isShowDelete.value = false;
      fetchPortPools();
    } finally {
      isDeleting.value = false;
    }
  }

  /** 确认提交端口池 */
  async function handleConfirmPortPool(data: ConfirmData) {
    const { poolItems } = data;
    isSubmitting.value = true;
    try {
      if (isEdit.value) {
        await PortPoolService.updatePortPool({
          envID: props.envId,
          name: editData.value?.name || '',
          poolItems,
        });
      } else {
        await PortPoolService.createPortPool({
          envID: props.envId,
          name: data.name,
          poolItems,
        });
      }
      setTimeout(() => {
        Message({ message: t('操作成功'), theme: 'success' });
        isShowSideslider.value = false;
        fetchPortPools();
      }, 500);
    } finally {
      setTimeout(() => {
        isSubmitting.value = false;
      }, 500);
    }
  }

  /** 新建端口池 */
  function handleCreatePortPool() {
    isEdit.value = false;
    editData.value = null;
    isShowSideslider.value = true;
  }

  /** 删除端口池 - 打开确认弹窗 */
  function handleDeletePortPool(row: PortPoolConfigOutputObj) {
    deleteTarget.value = row;
    isShowDelete.value = true;
  }

  /** 编辑端口池 */
  function handleEditPortPool(row: PortPoolConfigOutputObj) {
    isEdit.value = true;
    editData.value = row;
    isShowSideslider.value = true;
  }

  function handlePageBack() {
    const activeEnvName = (route.query.envName as string) || '';
    router.push({
      name: 'env',
      query: activeEnvName ? { active: activeEnvName } : undefined,
    });
  }

  function handlePageLimitChange(val: number) {
    pagination.value.limit = val;
    pagination.value.current = 1;
  }

  function handlePageValueChange(val: number) {
    pagination.value.current = val;
  }

  const { start: startPoll } = useInterval(fetchPortPools, 10_000);

  onMounted(async () => {
    await fetchEnvDetail();
    fetchPortPools();
    startPoll();
  });
</script>

<style lang="postcss" scoped>
  :deep(.bk-modal-body) {
    .bk-modal-header {
      .bk-dialog-header {
        padding-top: 48px;
      }
    }
    .bk-modal-footer {
      .bk-dialog-footer {
        border: none;
        background-color: unset;
        padding-top: 0;
        padding-bottom: 24px;
      }
    }
  }
</style>
