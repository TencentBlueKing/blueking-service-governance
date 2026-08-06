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
  <div class="port-config-table">
    <Form
      ref="formRef"
      form-type="vertical"
      :model="portList"
    >
      <Table
        ref="tableRef"
        class="w-full"
        :data="[]"
        row-class-name="group"
        :row-config="{
          isHover: true,
        }"
        :row-height="43"
        show-overflow-tooltip
        v-bind="$attrs"
      >
        <template #empty>
          <TableException />
        </template>
        <!-- 端口名称 -->
        <TableColumn
          :label="$t('端口名称')"
          min-width="120"
          :required="true"
        >
          <template #default="{ row, rowIndex }: { row: PortItem; rowIndex?: number }">
            <Form.FormItem
              class="mb-0"
              error-display-type="tooltips"
              label=""
              :property="`${rowIndex}.name`"
              :rules="rules.name"
            >
              <Input
                ref="nameInputRef"
                v-model.trim="row.name"
                class="w-full relative z-99"
                clearable
                :placeholder="$t('请输入')"
              />
            </Form.FormItem>
          </template>
        </TableColumn>

        <!-- 监听端口 -->
        <TableColumn
          :label="$t('监听端口')"
          min-width="120"
          :required="true"
        >
          <template #default="{ row, rowIndex }: { row: PortItem; rowIndex?: number }">
            <Form.FormItem
              class="mb-0 port-item"
              error-display-type="tooltips"
              :icon-offset="100"
              label=""
              :property="`${rowIndex}.port`"
              :rules="rules.port"
            >
              <Input
                v-model.number="row.port"
                class="w-full relative z-99"
                :clearable="false"
                :placeholder="$t('请输入')"
                type="number"
              />
            </Form.FormItem>
          </template>
        </TableColumn>

        <!-- 协议 -->
        <TableColumn
          :label="$t('协议')"
          min-width="120"
          :required="true"
        >
          <template #default="{ row, rowIndex }: { row: PortItem; rowIndex?: number }">
            <Form.FormItem
              class="mb-0"
              error-display-type="tooltips"
              label=""
              :property="`${rowIndex}.protocol`"
            >
              <Select
                v-model="row.protocol"
                class="w-full relative z-99"
                :clearable="false"
                :placeholder="$t('请选择')"
              >
                <Select.Option
                  v-for="item in protocolOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </Select>
            </Form.FormItem>
          </template>
        </TableColumn>

        <!-- 目标端口 -->
        <TableColumn
          :label="$t('目标端口')"
          min-width="120"
          :required="true"
        >
          <template #default="{ row, rowIndex }: { row: PortItem; rowIndex?: number }">
            <Form.FormItem
              class="mb-0"
              error-display-type="tooltips"
              label=""
              :property="`${rowIndex}.targetPort`"
              required
            >
              <Input
                v-model.number="row.targetPort"
                class="w-full relative z-99"
                :clearable="false"
                :placeholder="$t('请输入')"
              />
            </Form.FormItem>
          </template>
        </TableColumn>

        <!-- 操作列 -->
        <TableColumn
          :label="$t('操作')"
          width="100"
        >
          <template #default="{ row }">
            <div class="h-[32px] flex items-center gap-[8px]">
              <!-- 添加行 -->
              <i
                class="bkms-icon bkms-icon-plus-circle-shape cursor-pointer text-[#C4C6CC] hover:text-[#4D4F56]"
                @click="addPort"
              ></i>
              <!-- 删除行 -->
              <template v-if="portList.length > 1">
                <i
                  class="bkms-icon bkms-icon-minus-circle-shape cursor-pointer text-[#C4C6CC] hover:text-[#4D4F56]"
                  @click="deletePort(row)"
                ></i>
              </template>
              <template v-else>
                <i
                  v-bk-tooltips="$t('至少保留一个')"
                  class="bkms-icon bkms-icon-minus-circle-shape cursor-not-allowed text-[#DCDEE5]"
                ></i>
              </template>
            </div>
          </template>
        </TableColumn>
      </Table>
    </Form>
  </div>
</template>

<script lang="ts" setup>
  import { ref, watch } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Form, Input, Select } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { BKMS_REGEX } from '~/common/const';

  // 端口配置接口
  export interface PortConfig {
    // 端口名称
    name: string;
    // 监听端口
    port: number | string;
    // 协议
    protocol: string;
    // 目标端口
    targetPort: number | string;
  }

  const props = defineProps<{
    // 端口列表
    list?: PortConfig[];
  }>();

  const emit = defineEmits<{
    update: [value: PortConfig[]];
  }>();

  const { t } = useI18n();

  // 协议选项
  const protocolOptions = [
    { label: 'TCP', value: 'TCP' },
    { label: 'UDP', value: 'UDP' },
  ];

  // 内部使用的端口项类型（带唯一ID）
  interface PortItem extends PortConfig {
    id: string; // 唯一标识
  }

  // 表单校验规则
  const rules = ref({
    name: [
      {
        validator: (value: string) => {
          if (!value || value.trim() === '') {
            return false;
          }
          return true;
        },
        message: t('端口名称不能为空'),
        trigger: 'blur',
      },
      {
        pattern: BKMS_REGEX.serviceNameRegex,
        message: t('以小写字母开头，只能包含小写字母、数字和连字符，长度 1-63 个字符'),
        trigger: 'blur',
      },
      {
        validator: (value: string) => {
          if (!value) return true;
          // 检查是否有重复的端口名称
          const duplicates = portList.value.filter(item => item.name === value);
          return duplicates.length <= 1;
        },
        message: t('端口名称不能重复'),
        trigger: 'blur',
      },
    ],
    port: [
      {
        validator: (value: number | string) => {
          const port = Number(value);
          return !isNaN(port) && port > 0 && port <= 65535;
        },
        message: t('端口范围为 1-65535'),
        trigger: 'blur',
      },
    ],
  });

  // 数据列表
  const portList = ref<PortItem[]>([]);

  // 标记是否正在同步 props，防止循环更新
  const isSyncingFromProps = ref(false);

  // 监听props变化
  watch(
    () => props.list,
    newList => {
      isSyncingFromProps.value = true;
      const list = newList || [];

      // 如果列表为空，添加一行默认数据
      if (list.length === 0) {
        portList.value = [
          {
            id: generateId(),
            name: '',
            port: '',
            protocol: 'TCP',
            targetPort: '',
          },
        ];
      } else {
        portList.value = list.map(item => ({
          ...item,
          id: generateId(),
        }));
      }
      // 在下一个 tick 后重置标记
      setTimeout(() => {
        isSyncingFromProps.value = false;
      }, 0);
    },
    { immediate: true, deep: true },
  );

  // 监听 portList 变化，触发 update 事件
  watch(
    portList,
    () => {
      // 如果是从 props 同步过来的变化，不触发 update 事件
      if (isSyncingFromProps.value) {
        return;
      }
      emit('update', portList.value);
    },
    { deep: true },
  );

  // 生成唯一ID
  function generateId(): string {
    return Date.now().toString() + Math.random().toString(36).substring(2, 11);
  }

  // 表格验证
  const formRef = ref<InstanceType<typeof Form>>();

  // 添加端口
  function addPort() {
    const newPort: PortItem = {
      id: generateId(),
      name: '',
      port: '',
      protocol: 'TCP', // 默认TCP
      targetPort: '',
    };

    portList.value.push(newPort);
  }

  // 删除端口
  function deletePort(row: PortItem) {
    // 至少保留一行
    if (portList.value.length <= 1) {
      return;
    }

    const index = portList.value.findIndex(item => item.id === row.id);
    if (index > -1) {
      portList.value.splice(index, 1);
    }
  }

  async function tableValidate() {
    try {
      const valid = await formRef.value?.validate();
      return valid;
    } catch {
      return false;
    }
  }

  defineExpose({
    tableValidate,
  });
</script>

<style lang="postcss" scoped>
  .port-config-table {
    :deep(.bk-form-error-tips) {
      z-index: 999;
    }
    .port-item :deep(.bk-form-error-tips) {
      right: 28px;
    }
  }
</style>
