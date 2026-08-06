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
    :width="800"
    @hidden="handleHidden"
    @shown="handleShown"
  >
    <template #header>
      <div class="flex items-center">
        <span>{{ isEdit ? $t('编辑服务') : $t('新建服务') }}（Service）</span>
        <template v-if="isEdit">
          <Divider
            class="mx-[12px]"
            direction="vertical"
          ></Divider>
          <span class="text-[#979BA5] text-[14px] leading-[24px]">{{ currentService?.name }}</span>
        </template>
      </div>
    </template>
    <div class="px-[24px] pt-[18px]">
      <Form
        ref="formRef"
        form-type="vertical"
        :model="formData"
        :rules="formRules"
      >
        <Form.FormItem
          :label="$t('服务名称')"
          property="name"
          required
        >
          <Input
            v-model.trim="formData.name"
            :disabled="isEdit"
            :placeholder="$t('请输入 1-63 个字符服务名称，只能包含小写字母、数字和连字符，以小写字母开头')"
          />
        </Form.FormItem>
        <Form.FormItem
          :label="$t('选择器')"
          required
        >
          <KeyValue
            ref="keyValueRef"
            v-model="formData.selector"
            key-placeholder="请输入 Key"
            :key-rules="selectorKeyRules"
            :min-rows="1"
            textarea
            value-placeholder="请输入 Value"
            @init:model-value="handleInitKeyValue"
          />
        </Form.FormItem>
        <!-- 第一期：没有设置泳道服务，默认为 true -->
        <!-- <Form.FormItem :label="$t('泳道服务')">
          <Switcher
            v-model="formData.trafficLaneEnabled"
            theme="primary"
          />
        </Form.FormItem> -->
        <Form.FormItem
          :label="$t('端口配置')"
          required
        >
          <PortConfigTable
            ref="portConfigTableRef"
            :list="formData.ports"
            @update="handlePortsUpdate"
          />
        </Form.FormItem>
      </Form>
    </div>

    <template #footer>
      <div class="flex items-center">
        <Button
          class="mr-[8px]"
          :loading="loading"
          theme="primary"
          @click="handleSubmit"
        >
          {{ isEdit ? $t('保存') : $t('确定') }}
        </Button>
        <Button @click="handleCancel">
          {{ $t('取消') }}
        </Button>
      </div>
    </template>
  </Sideslider>
</template>

<script setup lang="ts">
  import { computed, reactive, ref, watch } from 'vue';

  import { Button, Divider, Form, Input, Sideslider } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { BKMS_REGEX } from '~/common/const';
  import KeyValue, { type FormRule } from '~/components/key-value.vue';
  import useLeaveConfirm from '~/composables/use-leave-confirm';
  import { useAppDetail } from '~/stores/app-detail';

  import PortConfigTable, { type PortConfig } from './port-config-table.vue';

  interface Emits {
    (e: 'update:visible', value: boolean): void;
    (e: 'submit', data: ServiceFormData): void;
    (e: 'cancel'): void;
  }

  interface Props {
    currentService?: null | ServiceFormData;
    isEdit: boolean;
    loading?: boolean;
    visible: boolean;
  }

  interface ServiceFormData {
    name: string;
    ports: PortConfig[];
    selector: Array<{ key: string; value: string }>;
    trafficLaneEnabled: boolean;
  }

  const props = withDefaults(defineProps<Props>(), {
    loading: false,
    currentService: null,
  });

  const emit = defineEmits<Emits>();

  const { t } = useI18n();
  const appDetailStore = useAppDetail();

  const formRef = ref<InstanceType<typeof Form>>();
  const portConfigTableRef = ref<InstanceType<typeof PortConfigTable>>();
  const keyValueRef = ref();

  // 表单数据
  const defaultFormData: ServiceFormData = {
    name: '',
    selector: [{ key: 'app', value: appDetailStore.app }],
    ports: [],
    trafficLaneEnabled: true,
  };

  const formData = reactive<ServiceFormData>({ ...defaultFormData });
  const { confirmBox, withPausedWatch, forceCleanDirtyTag } = useLeaveConfirm(formData);

  // 表单验证规则
  const formRules = {
    name: [
      {
        pattern: BKMS_REGEX.serviceNameRegex,
        message: t('以小写字母开头，只能包含小写字母、数字和连字符，长度 1-63 个字符'),
        trigger: 'blur',
      },
    ],
  };

  // 选择器 key 校验规则
  const selectorKeyRules: FormRule[] = [
    {
      validator: (val: unknown) => {
        if (!val) return true;
        const strVal = val as string;
        const keys = formData.selector.filter(item => item && item.key).map(item => item.key);
        const duplicateCount = keys.filter(k => k === strVal).length;
        return duplicateCount <= 1;
      },
      message: t('Key 不能重复'),
      trigger: 'blur',
    },
  ];

  const visible = computed({
    get: () => props.visible,
    set: value => emit('update:visible', value),
  });

  // 监听弹窗显示状态，弹窗打开且非编辑模式时重置表单
  watch(
    () => props.visible,
    newVisible => {
      if (newVisible && !props.isEdit) {
        withPausedWatch(() => {
          resetForm();
        });
      }
    },
    { immediate: true },
  );

  // 侧边栏关闭前确认
  function handleBeforeClose(): Promise<boolean> {
    return confirmBox();
  }

  async function handleCancel() {
    if (await handleBeforeClose()) {
      emit('cancel');
    }
  }

  // 侧边栏隐藏时重置表单
  function handleHidden() {
    withPausedWatch(() => {
      resetForm();
    });
  }

  /**
   * @description 初始化会update两次,这里的formData.selector不好用watch + once清除DirtyTag
   * 故选择让KeyValue抛出init方法，作为初始化成功的钩子，从而清除formChange的dirtyTag
   */
  function handleInitKeyValue() {
    forceCleanDirtyTag();
  }

  // 端口配置更新
  function handlePortsUpdate(ports: PortConfig[]) {
    formData.ports = ports;
  }

  function handleShown() {
    // 如果是编辑模式，回填表单数据
    if (props.currentService && props.isEdit) {
      const serviceData: ServiceFormData = {
        name: props.currentService.name || '',
        selector: props.currentService.selector || [],
        ports: props.currentService.ports || [],
        trafficLaneEnabled: props.currentService.trafficLaneEnabled || false,
      };
      withPausedWatch(() => {
        Object.assign(formData, serviceData);
      });
    }
  }

  // 提交表单
  async function handleSubmit() {
    const valid = await formRef.value?.validate().catch(() => false);
    if (!valid) return;

    // 验证选择器（使用 KeyValue 组件的校验）
    const selectorValid = await keyValueRef.value?.validate().catch(() => false);
    if (!selectorValid) return;

    // 验证端口配置表格
    const portValid = await portConfigTableRef.value?.tableValidate().catch(() => false);
    if (!portValid) return;

    // 验证至少有一个有效端口
    const validPorts = formData.ports.filter(port => port.name && port.port && port.protocol && port.targetPort);
    if (validPorts.length === 0) {
      console.warn('至少需要配置一个完整的端口');
      return;
    }

    // 验证至少有一个选择器
    const validSelectors = formData.selector.filter(item => item.key && item.value);
    if (validSelectors.length === 0) {
      console.warn('至少需要配置一个选择器');
      return;
    }
    forceCleanDirtyTag(() => {
      emit('submit', { ...formData });
    });
  }

  // 重置表单数据和验证状态
  function resetForm() {
    Object.assign(formData, JSON.parse(JSON.stringify(defaultFormData)));
    formRef.value?.clearValidate();
    keyValueRef.value?.clearValidate();
  }
</script>
