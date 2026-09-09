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
  <Sideslider
    v-model:is-show="isShow"
    render-directive="if"
    :width="720"
  >
    <template #header>
      <DividerHeader
        :title="$t('配置仪表盘')"
        :title-size="16"
      >
        <span class="truncate text-[#979BA5]">{{ appDetailStore.app || '--' }}</span>
      </DividerHeader>
    </template>

    <div class="flex h-full flex-col gap-[16px] px-[24px] py-[20px]">
      <Alert theme="info">
        <DashboardSetupGuide compact />
      </Alert>

      <div>
        <Button
          theme="primary"
          @click="openCreateDialog"
        >
          <Plus
            class="mr-[4px]"
            :height="24"
            :width="24"
          />
          {{ $t('添加仪表盘') }}
        </Button>
      </div>

      <div class="min-h-0 flex-1">
        <Table
          v-bkloading="{ loading, zIndex: 6 }"
          :data="curPageData"
          :pagination="pagination"
          :row-config="{ isHover: true }"
          @page-limit-change="pageSizeChange"
          @page-value-change="pageChange"
        >
          <template #empty>
            <TableException type="empty" />
          </template>
          <TableColumn
            field="title"
            :label="$t('仪表盘名称')"
            min-width="200"
            show-overflow="tooltip"
          />
          <TableColumn
            fixed="right"
            :label="$t('操作')"
            :width="120"
          >
            <template #default="{ row }">
              <div class="flex items-center gap-[12px]">
                <Button
                  text
                  theme="primary"
                  @click="handleEdit(row)"
                >
                  {{ $t('编辑') }}
                </Button>
                <Button
                  text
                  theme="primary"
                  @click="handleDelete(row)"
                >
                  {{ $t('删除') }}
                </Button>
              </div>
            </template>
          </TableColumn>
        </Table>
      </div>
    </div>
  </Sideslider>

  <Dialog
    v-model:is-show="dialogVisible"
    :quick-close="false"
    :title="isEdit ? $t('编辑仪表盘') : $t('添加仪表盘')"
    width="480"
  >
    <Form
      ref="formRef"
      form-type="vertical"
      :model="formData"
      :rules="formRules"
    >
      <Form.FormItem
        :label="$t('仪表盘名称')"
        property="uid"
        required
      >
        <div class="flex items-center gap-[6px]">
          <Select
            v-model="formData.uid"
            class="min-w-0 flex-1"
            filterable
            :loading="monitorDashboardsLoading"
            :placeholder="$t('请选择监控平台中的仪表盘')"
            @change="handleUidChange"
          >
            <Select.Option
              v-for="item in catalogOptions"
              :key="item.uid"
              :disabled="isUidTaken(item.uid)"
              :label="item.title || item.uid"
              :value="item.uid"
            >
              <div class="flex w-full min-w-0 items-center justify-between gap-[8px] leading-[20px]">
                <OverflowTitle
                  class="min-w-0 flex-1"
                  type="tips"
                >
                  {{ item.title || item.uid }}
                </OverflowTitle>
                <span
                  v-if="isUidTaken(item.uid)"
                  class="shrink-0 text-[12px] text-[#979BA5]"
                >
                  {{ $t('已添加到本应用') }}
                </span>
              </div>
            </Select.Option>
            <template #extension>
              <Button
                class="w-full"
                text
                theme="primary"
                @click="handleCreateInMonitor"
              >
                <Plus
                  class="mr-[4px]"
                  :height="16"
                  :width="16"
                />
                {{ $t('前往监控平台新建') }}
                <Share class="ml-[4px]" />
              </Button>
            </template>
          </Select>
          <Button
            v-bk-tooltips="$t('刷新仪表盘列表')"
            class="!min-w-[32px] !px-0"
            :disabled="monitorDashboardsLoading"
            text
            theme="primary"
            @click="handleRefreshCatalog"
          >
            <i
              :class="[
                'bkms-icon bkms-icon-refresh text-[16px]',
                monitorDashboardsLoading ? 'animate-spin [animation-duration:1.5s]' : '',
              ]"
            ></i>
          </Button>
        </div>
        <div class="mt-[4px] text-[12px] text-[#979BA5]">
          {{ $t('从监控平台已有仪表盘中选择，新建后可刷新列表') }}
        </div>
      </Form.FormItem>
    </Form>
    <template #footer>
      <Button
        :loading="submitting"
        theme="primary"
        @click="handleSubmit"
      >
        {{ $t('确定') }}
      </Button>
      <Button
        class="ml-[8px]"
        @click="dialogVisible = false"
        >{{ $t('取消') }}
      </Button>
    </template>
  </Dialog>
</template>

<script setup lang="ts">
  import { computed, h, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Alert, Button, Dialog, Form, InfoBox, Message, OverflowTitle, Select, Sideslider } from 'bkui-vue';
  import { Plus, Share } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import DividerHeader from '~/components/divider-header.vue';
  import TableException from '~/components/table-exception.vue';
  import usePageConf from '~/composables/use-page';
  import useToLink from '~/composables/use-to-link';
  import { useAppDetail } from '~/stores/app-detail';
  import { useSpaceStore } from '~/stores/space';

  import DashboardSetupGuide from './dashboard-setup-guide.vue';
  import { type AppDashboardBinding, type MonitorDashboardOption, useAppDashboards } from './use-app-dashboards';

  const isShow = defineModel<boolean>('isShow', { default: false });
  const emit = defineEmits<{
    (e: 'updated', uid?: string): void;
  }>();

  const { t } = useI18n();
  const spaceStore = useSpaceStore();
  const appDetailStore = useAppDetail();
  const { handleToLink } = useToLink();
  const {
    createDashboard,
    dashboards,
    deleteDashboard,
    fetchDashboards,
    fetchMonitorDashboards,
    loading,
    monitorDashboards,
    monitorDashboardsLoading,
    updateDashboard,
  } = useAppDashboards();

  const dialogVisible = ref(false);
  const submitting = ref(false);
  const editingRow = ref<AppDashboardBinding>();
  const formRef = ref<InstanceType<typeof Form> | null>(null);
  const formData = ref({
    title: '',
    uid: '',
  });

  const isEdit = computed(() => Boolean(editingRow.value));
  /** 可选仪表盘：补齐当前选中项（可能已不在监控列表中），并把已绑定的排到末尾 */
  const catalogOptions = computed<MonitorDashboardOption[]>(() => {
    const items = [...monitorDashboards.value];
    const currentUid = formData.value.uid;
    if (currentUid && !items.some(item => item.uid === currentUid)) {
      items.unshift({
        folderTitle: '',
        title: formData.value.title || currentUid,
        uid: currentUid,
        url: '',
      });
    }
    return items.sort((a, b) => Number(isUidTaken(a.uid)) - Number(isUidTaken(b.uid)));
  });
  const formRules = {
    uid: [{ message: t('请选择仪表盘名称'), required: true, trigger: 'change' }],
  };
  const { curPageData, pagination, pageChange, pageSizeChange, handleResetPage } = usePageConf(
    dashboards,
    {
      current: 1,
      limit: 10,
      remote: false,
    },
    computed(() => dashboards.value.length),
  );

  /** 跳到监控平台新建仪表盘，缺少监控业务 ID 时给出提示 */
  function handleCreateInMonitor() {
    const bizId = spaceStore.workspaceDetail?.bkSystems?.bkMonitorProjectID;
    if (!bizId) {
      Message({ message: t('当前空间未配置蓝鲸监控业务，无法跳转'), theme: 'warning' });
      return;
    }
    handleToLink('monitor-grafana', bizId);
  }

  /** 删除：二次确认后调用接口并通知父组件刷新 */
  function handleDelete(row: AppDashboardBinding) {
    if (!appDetailStore.appID || !row.uid) return;
    InfoBox({
      cancelText: t('取消'),
      confirmButtonTheme: 'danger',
      confirmText: t('删除'),
      content: h('div', { class: 'text-left' }, [
        h('div', [t('展示名称: {name}', { name: row.title || row.uid })]),
        h('div', { class: 'mt-[14px] bg-[#F5F7FA] py-[12px] px-[16px]' }, [t('删除后，将不可恢复，请谨慎操作！')]),
      ]),
      footerAlign: 'center',
      headerAlign: 'center',
      title: t('确认删除该仪表盘？'),
      async onConfirm() {
        if (!appDetailStore.appID || !row.uid) return;
        await deleteDashboard(appDetailStore.appID, row.uid);
        Message({ message: t('删除成功'), theme: 'success' });
        emit('updated');
      },
    });
  }

  /** 编辑：回填当前绑定信息并以编辑模式打开弹窗 */
  function handleEdit(row: AppDashboardBinding) {
    editingRow.value = row;
    formData.value = {
      title: row.title || '',
      uid: row.uid || '',
    };
    dialogVisible.value = true;
  }

  /** 手动刷新监控平台仪表盘列表 */
  async function handleRefreshCatalog() {
    await fetchMonitorDashboards(spaceStore.currentSpace);
    Message({ message: t('仪表盘列表已刷新'), theme: 'success' });
  }

  /** 提交：展示名称留空时依次回退为监控平台标题、仪表盘 uid */
  async function handleSubmit() {
    const valid = await formRef.value?.validate().catch(() => false);
    if (!valid || !appDetailStore.appID || !formData.value.uid) return;

    submitting.value = true;
    try {
      const payload = {
        title:
          formData.value.title ||
          catalogOptions.value.find(item => item.uid === formData.value.uid)?.title ||
          formData.value.uid,
        uid: formData.value.uid,
      };
      if (editingRow.value?.uid) {
        await updateDashboard(appDetailStore.appID, editingRow.value.uid, payload);
        Message({ message: t('更新成功'), theme: 'success' });
      } else {
        await createDashboard(appDetailStore.appID, payload);
        Message({ message: t('创建成功'), theme: 'success' });
        handleResetPage();
      }
      dialogVisible.value = false;
      emit('updated', payload.uid);
    } finally {
      submitting.value = false;
    }
  }

  /** 切换仪表盘时同步展示名称，接口仍需传 title */
  function handleUidChange(uid: string) {
    const selected = catalogOptions.value.find(item => item.uid === uid);
    formData.value.title = selected?.title || uid;
  }

  /** 该仪表盘是否已被本应用绑定（编辑时排除自身） */
  function isUidTaken(uid: string) {
    return dashboards.value.some(item => item.uid === uid && item.uid !== editingRow.value?.uid);
  }

  /** 新建：清空表单并以新增模式打开弹窗 */
  function openCreateDialog() {
    editingRow.value = undefined;
    resetForm();
    dialogVisible.value = true;
  }

  function resetForm() {
    formData.value = { title: '', uid: '' };
    formRef.value?.clearValidate();
  }

  defineExpose({ openCreateDialog });

  // 侧栏打开时重置分页并拉取当前应用的绑定列表
  watch(
    [() => appDetailStore.appID, isShow],
    ([appID, show]) => {
      if (show && appID) {
        handleResetPage();
        fetchDashboards(appID);
      }
    },
    { immediate: true },
  );

  // 弹窗打开时才加载监控平台仪表盘目录，避免无谓请求
  watch(dialogVisible, visible => {
    if (visible && spaceStore.currentSpace) {
      fetchMonitorDashboards(spaceStore.currentSpace);
    }
  });
</script>
