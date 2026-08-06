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
    v-model:is-show="visible"
    :before-close="handleBeforeClose"
    quick-close
    render-directive="if"
    :width="960"
    @hidden="handleHidden"
    @shown="handleShown"
  >
    <template #header>
      <div class="flex items-center">
        <span>{{ isEdit ? $t('编辑端口池') : $t('新建端口池') }}</span>
        <template v-if="isEdit">
          <Divider
            class="mx-[12px]"
            direction="vertical"
          ></Divider>
          <span class="text-[#979BA5] text-[14px]">{{ editData?.name || '--' }}</span>
        </template>
      </div>
    </template>
    <div
      class="px-[24px] pt-[18px] overflow-y-auto"
      style="max-height: calc(100vh - 120px)"
    >
      <Form
        ref="formRef"
        form-type="vertical"
        label-position="top"
        :model="formData"
        :rules="formRules"
      >
        <ToggleCard
          :name="$t('基本信息')"
          type="normal"
        >
          <div class="flex gap-[16px]">
            <Form.FormItem
              class="flex-1"
              :label="$t('端口池名称')"
              property="name"
              :required="true"
            >
              <Input
                v-model.trim="formData.name"
                :disabled="isEdit"
                :placeholder="$t('由小写字母、数字、- 组成，以字母开头且字母数字结尾，最多63字符')"
              />
            </Form.FormItem>
            <Form.FormItem
              class="flex-1"
              :label="$t('协议')"
              property="protocol"
              :required="true"
            >
              <Select
                v-model="formData.protocol"
                :clearable="false"
                :disabled="isEdit"
                :list="protocolOptions"
              />
            </Form.FormItem>
          </div>

          <div class="flex gap-[16px]">
            <Form.FormItem
              class="flex-1"
              :label="$t('起始端口')"
              property="startPort"
              :required="true"
            >
              <Input
                v-model.number="formData.startPort"
                :disabled="isEdit"
                :placeholder="$t('请输入 1-65535 之间的数字')"
                type="number"
                @change="() => formRef?.validate?.('endPort')"
              />
            </Form.FormItem>
            <Form.FormItem
              class="flex-1"
              :label="$t('结束端口')"
              property="endPort"
              :required="true"
            >
              <Input
                v-model.number="formData.endPort"
                :placeholder="$t('请输入 1-65535 之间的数字')"
                type="number"
              />
            </Form.FormItem>
          </div>

          <Form.FormItem
            class="!mb-[0px]"
            :label="$t('端口段长度')"
            property="segmentLength"
            :required="true"
          >
            <Input
              v-model.number="formData.segmentLength"
              :disabled="isEdit"
              type="number"
            />
            <template #description>
              <span class="text-[#979BA5] text-[12px]">{{
                $t('端口数量需要与 Agones 组件中端口数量的配置保持一致')
              }}</span>
            </template>
          </Form.FormItem>
        </ToggleCard>

        <!-- 负载均衡配置 -->
        <ToggleCard
          class="mt-[24px]"
          :name="$t('负载均衡配置')"
          type="normal"
        >
          <Form
            ref="lbFormRef"
            form-type="vertical"
            :model="loadBalancerList"
          >
            <Table
              class="lb-table"
              :data="loadBalancerList"
              :row-config="{ isHover: true, height: 44 }"
            >
              <TableColumn
                field="ids"
                min-width="300"
              >
                <template #header>
                  {{ $t('负载均衡 ID') }} <span style="color: #ea3636">*</span>
                  <i
                    v-bk-tooltips="{ content: lbIdFormatTip, placement: 'top' }"
                    class="bkms-icon bkms-icon-circle-info ml-[6px] text-[14px] text-[#63656e]"
                  ></i>
                </template>
                <template #default="{ row, rowIndex }: { row: any; rowIndex?: number }">
                  <!-- 错误 icon 位置调整 -->
                  <Form.FormItem
                    class="mb-0 custom-form-item"
                    :class="{ 'has-tag-value': row.ids?.length }"
                    error-display-type="tooltips"
                    label=""
                    :property="`${rowIndex}.ids`"
                    :rules="lbRules.ids"
                  >
                    <TagInput
                      v-model="row.ids"
                      allow-create
                      has-delete-icon
                      :placeholder="$t('输入后按回车添加')"
                      @change="handleLbIdsChange"
                    />
                    <template #error>xxx55</template>
                  </Form.FormItem>
                </template>
              </TableColumn>
              <TableColumn
                field="external"
                :label="$t('附加信息（external）')"
                min-width="240"
              >
                <template #default="{ row }: { row: LoadBalancerRow }">
                  <Input v-model.trim="row.external" />
                </template>
              </TableColumn>
              <TableColumn
                :label="$t('操作')"
                width="80"
              >
                <template #default="{ rowIndex }: { row: any; rowIndex?: number }">
                  <Button
                    text
                    theme="primary"
                    @click="handleRemoveRow(rowIndex!)"
                  >
                    {{ $t('删除') }}
                  </Button>
                </template>
              </TableColumn>
            </Table>
          </Form>

          <!-- 添加行 -->
          <Button
            class="mt-[12px]"
            text
            theme="primary"
            @click="handleAddRow"
          >
            <i class="bkms-icon bkms-icon-plus-circle-shape mr-[5px]"></i>
            {{ $t('添加') }}
          </Button>
        </ToggleCard>
      </Form>
    </div>
    <template #footer>
      <div class="flex items-center">
        <Button
          class="mr-[8px]"
          :loading="props.loading"
          theme="primary"
          @click="handleConfirm"
        >
          {{ $t('确定') }}
        </Button>
        <Button
          :disabled="props.loading"
          @click="handleCancel"
        >
          {{ $t('取消') }}
        </Button>
      </div>
    </template>
  </Sideslider>
</template>

<script setup lang="ts">
  import { computed, reactive, ref } from 'vue';

  import { Table, TableColumn } from '@blueking/table';
  import { Button, Divider, Form, Input, Select, Sideslider, TagInput } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { PortPoolConfigOutputObj, PortPoolItemInput } from '~/@types/v1/port-pool';
  import useLeaveConfirm from '~/composables/use-leave-confirm';

  interface ConfirmData {
    name: string;
    poolItems: PortPoolItemInput[];
  }

  interface Emits {
    (e: 'cancel'): void;
    (e: 'confirm', data: ConfirmData): void;
    (e: 'update:visible', value: boolean): void;
  }

  interface LoadBalancerRow {
    external: string;
    ids: string[];
  }

  interface Props {
    editData?: null | PortPoolConfigOutputObj;
    isEdit?: boolean;
    loading?: boolean;
    visible: boolean;
  }

  const props = withDefaults(defineProps<Props>(), {
    editData: null,
    isEdit: false,
    loading: false,
  });

  const emit = defineEmits<Emits>();
  const { t } = useI18n();

  const visible = computed({
    get: () => props.visible,
    set: value => emit('update:visible', value),
  });

  // 表单引用
  const formRef = ref();
  const lbFormRef = ref();

  // 协议选项
  const protocolOptions = [
    { label: 'TCP', value: 'TCP' },
    { label: 'UDP', value: 'UDP' },
  ];

  // 表单数据
  const formData = reactive<{
    endPort: null | number;
    name: string;
    protocol: string;
    segmentLength: null | number;
    startPort: null | number;
  }>({
    name: '',
    protocol: 'TCP',
    startPort: null,
    endPort: null,
    segmentLength: null,
  });

  // 使用 useLeaveConfirm hook 管理表单变化检测
  const { confirmBox, withPausedWatch, forceCleanDirtyTag } = useLeaveConfirm(formData);

  // 端口范围校验器
  const portRangeValidator = (label: string) => [
    {
      validator: (value: number | string) => {
        if (value == null || value === '') return true;
        const num = Number(value);
        if (!Number.isInteger(num) || num < 1 || num > 65535) {
          return t('{label}范围为 1-65535', { label });
        }
        return true;
      },
      trigger: 'blur',
    },
  ];

  // 结束端口 > 起始端口 校验
  const endPortGreaterValidator = {
    validator: () => {
      if (formData.startPort != null && formData.endPort != null && formData.endPort <= formData.startPort) {
        return t('结束端口必须大于起始端口');
      }
      return true;
    },
    trigger: 'blur',
  };

  // 编辑模式下结束端口只能改大
  const endPortNotDecreaseValidator = {
    validator: (value: number) => {
      if (!props.isEdit || !props.editData?.poolItems?.[0]?.endPort) return true;
      const originalEndPort = props.editData.poolItems[0].endPort;
      if (value != null && value < originalEndPort) {
        return t('修改端口池配置时，结束端口的值只能改大');
      }
      return true;
    },
    trigger: 'blur',
  };

  // 负载均衡 ID 格式校验: lb-xxxxxxxx / ap-xxx:lb-xxxxxxxx
  const LB_ID_REGEX = /^(lb-[a-z0-9]{8}|[a-z]{2}-[a-z]+:lb-[a-z0-9]{8})$/;

  // 负载均衡 ID 格式提示
  const lbIdFormatTip = computed(() =>
    [
      t('负载均衡 ID 格式说明：'),
      t(' - 必须以 "lb-" 开头'),
      t(' - 后接 8 位小写字母和数字组合'),
      t(' - 示例：lb-01auxtlh'),
      t(' - 可选：添加区域标识，格式为 <区域>:<ID>'),
      t(' - 示例：ap-shenzhen:lb-01auxtlh'),
    ].join('\n'),
  );

  // 端口池名称校验正则：最多63字符，只能包含小写字母、数字和 '-'，必须以字母开头，必须以字母数字结尾
  const PORT_POOL_NAME_REGEX = /^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$/;

  // 表单校验规则
  const formRules = {
    name: [
      {
        validator: (value: string) => {
          if (!value) return true;
          if (!PORT_POOL_NAME_REGEX.test(value)) {
            return t('由小写字母、数字、- 组成，以字母开头且字母数字结尾，最多63字符');
          }
          return true;
        },
        trigger: 'blur',
      },
    ],
    endPort: [...portRangeValidator(t('结束端口')), endPortGreaterValidator, endPortNotDecreaseValidator],
    startPort: [...portRangeValidator(t('起始端口'))],
  };

  // 负载均衡表格校验规则
  const lbRules = {
    ids: [
      { required: true, message: t('负载均衡 ID 不能为空'), trigger: 'change' },
      {
        validator: (value: string[]) => {
          if (!value || !value.length) return true;
          const invalid = value.find(id => !LB_ID_REGEX.test(id));
          if (invalid) {
            return lbIdFormatTip.value;
          }
          return true;
        },
        trigger: 'change',
      },
    ],
  };

  // 负载均衡配置列表
  const loadBalancerList = ref<LoadBalancerRow[]>([]);

  /** 添加一行负载均衡配置 */
  function handleAddRow() {
    loadBalancerList.value.push({ ids: [], external: '' });
  }

  /** 侧边栏关闭前确认 */
  function handleBeforeClose(): boolean | Promise<boolean> {
    return confirmBox();
  }

  async function handleCancel() {
    if (!(await confirmBox())) return;
    visible.value = false;
    emit('cancel');
  }

  async function handleConfirm() {
    try {
      await Promise.all([formRef.value?.validate(), lbFormRef.value?.validate()]);
    } catch {
      return;
    }
    // 组装 poolItems 数据
    const existingPoolItems = props.isEdit ? props.editData?.poolItems || [] : [];
    const poolItems: PortPoolItemInput[] = loadBalancerList.value.map((row, index) => ({
      itemName: existingPoolItems[index]?.itemName || '',
      protocol: formData.protocol,
      startPort: formData.startPort!,
      endPort: formData.endPort!,
      segmentLength: formData.segmentLength!,
      external: row.external,
      loadBalancerIDs: row.ids,
    }));

    emit('confirm', {
      name: formData.name,
      poolItems,
    });

    forceCleanDirtyTag();
  }

  // 侧栏关闭时重置状态
  function handleHidden() {
    resetForm();
  }

  /** 负载均衡 ID 变化时重新校验 */
  function handleLbIdsChange() {
    lbFormRef.value?.validate();
  }

  /** 删除一行负载均衡配置 */
  function handleRemoveRow(index: number) {
    loadBalancerList.value.splice(index, 1);
  }

  function handleShown() {
    if (props.isEdit && props.editData) {
      // 编辑模式：回填表单数据（暂停 watch 避免触发 dirty）
      withPausedWatch(() => {
        const firstItem = props.editData!.poolItems?.[0];
        formData.name = props.editData?.name || '';
        formData.protocol = firstItem?.protocol || 'TCP';
        formData.startPort = firstItem?.startPort ?? null;
        formData.endPort = firstItem?.endPort ?? null;
        formData.segmentLength = firstItem?.segmentLength ?? null;
        loadBalancerList.value = (props.editData!.poolItems || []).map(item => ({
          ids: item.loadBalancerIDs || [],
          external: item.external || '',
        }));
        if (!loadBalancerList.value.length) {
          loadBalancerList.value = [{ ids: [], external: '' }];
        }
      });
    } else {
      resetForm();
    }
  }

  /** 重置表单 */
  function resetForm() {
    withPausedWatch(() => {
      formData.name = '';
      formData.protocol = 'TCP';
      formData.startPort = null;
      formData.endPort = null;
      formData.segmentLength = null;
      loadBalancerList.value = [{ ids: [], external: '' }];
    });
    formRef.value?.clearValidate();
    lbFormRef.value?.clearValidate();
  }
</script>

<style lang="postcss" scoped>
  .lb-table :deep(.vxe-body--row .vxe-cell),
  .lb-table :deep(.vxe-header--row .vxe-cell) {
    padding: 0 10px !important;
  }

  /* 负载均衡 ID 校验失败时，TagInput 显示红色边框 */
  .lb-table :deep(.bk-form-item.is-error) .bk-tag-input .bk-tag-input-trigger {
    border-color: #ea3636;
  }

  .custom-form-item.has-tag-value :deep(.bk-form-error-tips) {
    right: 44px;
  }
</style>
