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
    <Alert
      class="mb-[16px]"
      closable
      theme="info"
      :title="$t('管理员能查看并执行“平台管理”导航下所有操作。')"
    />

    <div class="p-[16px] bg-[#fff] shadow-[0_2px_4px_0_#1919290d]">
      <div class="flex justify-between mb-[16px]">
        <Button
          theme="primary"
          @click="isShowAddAdministrator = true"
        >
          {{ t('添加平台管理员') }}
        </Button>
        <Input
          v-model="searchValue"
          class="w-[520px]"
          :placeholder="t('搜索管理员')"
        >
          <template #suffix>
            <Search class="text-[16px] text-[#979ba5] mr-[6px] mt-[2px]" />
          </template>
        </Input>
      </div>

      <!-- 管理员列表 -->
      <Table
        v-bkloading="{ loading: isLoading }"
        class="flex-1"
        :data="tableData"
        :pagination="pagination"
        :row-config="{
          isHover: true,
          isCurrent: true,
        }"
      >
        <template #empty>
          <TableException
            :type="curExceptionType"
            @clear="searchValue = ''"
            @refresh="fetchAdministratorList"
          />
        </template>
        <TableColumn
          field="username"
          :label="t('管理员')"
          min-width="220"
        >
          <template #default="{ row }">
            {{ row.username || '--' }}
          </template>
        </TableColumn>
        <TableColumn
          field="createdAt"
          :label="t('添加时间')"
          min-width="220"
          sortable
        >
          <template #default="{ row }">
            {{ row.createdAt ? formatDateString(row.createdAt) : '--' }}
          </template>
        </TableColumn>
        <TableColumn
          field="creator"
          :label="t('添加人')"
          min-width="220"
          sortable
        >
          <template #default="{ row }">
            {{ row.creator || '--' }}
          </template>
        </TableColumn>
        <TableColumn
          :label="t('操作')"
          width="180"
        >
          <template #default="{ row }">
            <PopConfirm
              :confirm-config="{
                theme: 'danger',
              }"
              confirm-text="确认删除"
              :title="t('确认删除该管理员？')"
              trigger="click"
              width="360"
              @confirm="handleDeleteAdministrator(row)"
            >
              <template #content>
                <div>{{ t('人员：{name}', { name: row.username }) }}</div>
                <div class="mt-[6px] mb-[16px]">{{ t('删除后，将不再具有该平台的相关权限，请谨慎操作！') }}</div>
              </template>
              <Button
                text
                theme="primary"
              >
                {{ t('删除') }}
              </Button>
            </PopConfirm>
          </template>
        </TableColumn>
      </Table>
    </div>

    <!-- 添加管理员弹窗 -->
    <Dialog
      v-model:is-show="isShowAddAdministrator"
      :quick-close="false"
      :title="t('添加平台管理员')"
      :width="500"
      @hidden="handleAddAdministratorHidden"
    >
      <Form
        ref="formRef"
        form-type="vertical"
        :model="addAdministratorForm"
        :rules="rules"
      >
        <Form.FormItem
          :label="t('管理员')"
          property="administrators"
          required
        >
          <UserSelector
            v-model="addAdministratorForm.administrators"
            multiple
          />
        </Form.FormItem>
      </Form>
      <template #footer>
        <Button
          :loading="isSubmitting"
          theme="primary"
          @click="handleAddAdministratorConfirm"
        >
          {{ t('确定') }}
        </Button>
        <Button
          class="ml-[8px]"
          @click="isShowAddAdministrator = false"
        >
          {{ t('取消') }}
        </Button>
      </template>
    </Dialog>
  </div>
</template>

<script lang="ts" setup>
  import { computed, nextTick, reactive, ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Alert, Button, Dialog, Form, Input, Message, PopConfirm } from 'bkui-vue';
  import { Search } from 'bkui-vue/lib/icon';
  import { useI18n } from 'vue-i18n';
  import { PlatmgtService } from '~/api/modules/v1';
  import UserSelector from '~/components/user-selector.vue';
  import useTableEmpty from '~/composables/use-table-empty';
  import useTime from '~/composables/use-time';

  import type { RoleBindingOutput } from '~/@types/v1/platmgt';

  const { t } = useI18n();
  const { formatDateString } = useTime();

  // 管理员列表相关状态
  const isLoading = ref(false);
  const administratorList = ref<RoleBindingOutput[]>([]);
  const pagination = ref({
    count: 0,
    current: 1,
    limit: 10,
  });

  const searchValue = ref('');
  const { setTypeToError, clearErrorType, curExceptionType } = useTableEmpty({
    filters: searchValue,
  });

  // 前端分页 + 搜索：根据搜索关键词过滤列表
  const tableData = computed(() =>
    administratorList.value.filter(item => {
      if (searchValue.value) {
        return (
          item.username?.toLowerCase().includes(searchValue.value.toLowerCase()) ||
          item.creator?.toLowerCase().includes(searchValue.value.toLowerCase())
        );
      }
      return true;
    }),
  );

  // 添加管理员弹窗相关状态
  const isShowAddAdministrator = ref(false);
  const isSubmitting = ref(false);
  const formRef = ref<InstanceType<typeof Form>>();
  const addAdministratorForm = reactive({
    administrators: [] as string[],
  });
  const rules = {
    administrators: [
      {
        required: true,
        message: t('请选择平台管理员'),
        trigger: 'change',
      },
    ],
  };

  // 获取管理员列表（一次性获取全部数据）
  async function fetchAdministratorList() {
    isLoading.value = true;
    try {
      administratorList.value = await PlatmgtService.listRoleBindings({});
      clearErrorType();
    } catch {
      setTypeToError();
    } finally {
      isLoading.value = false;
    }
  }

  // 添加管理员：表单校验通过后提交
  async function handleAddAdministratorConfirm() {
    const valid = await formRef.value?.validate().catch(() => false);
    if (!valid) return;

    isSubmitting.value = true;
    const isSuccess = await PlatmgtService.assignRoles({
      roleCode: 'admin',
      usernames: addAdministratorForm.administrators,
    })
      .then(() => true)
      .catch(() => false);
    isSubmitting.value = false;
    if (!isSuccess) return;

    Message({ message: t('添加成功'), theme: 'success' });
    isShowAddAdministrator.value = false;
    await fetchAdministratorList();
  }

  // 弹窗关闭后：重置表单并清除校验状态
  async function handleAddAdministratorHidden() {
    addAdministratorForm.administrators = [];
    await nextTick();
    formRef.value?.clearValidate();
  }

  // 删除管理员
  async function handleDeleteAdministrator(row: RoleBindingOutput) {
    if (!row.username) return;

    const isSuccess = await PlatmgtService.revokeRole({ username: row.username })
      .then(() => true)
      .catch(() => false);
    if (!isSuccess) return;

    Message({ message: t('删除成功'), theme: 'success' });
    await fetchAdministratorList();
  }

  // 搜索值变化时重置到第一页
  watch(searchValue, () => {
    pagination.value.current = 1;
  });

  // 过滤后的列表变化时更新总数
  watch(tableData, newValue => {
    pagination.value.count = newValue.length;
  });

  // 初始化加载
  fetchAdministratorList();
</script>
