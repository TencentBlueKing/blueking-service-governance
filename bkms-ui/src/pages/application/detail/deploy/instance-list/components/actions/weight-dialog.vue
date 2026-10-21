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
  <Dialog
    v-model:is-show="visible"
    :before-close="handleBeforeClose"
    :quick-close="false"
    render-directive="if"
    width="480"
  >
    <template #header>
      <DividerHeader :title="$t('调整权重')">
        <span class="truncate">
          {{ instanceIds.length === 1 ? instanceName : $t('已选 {0} 个实例', [instanceIds.length]) }}
        </span>
      </DividerHeader>
    </template>
    <Alert
      class="mb-[16px]"
      closable
      theme="info"
      :title="$t('权重为 0 时 该实例将不再接收流量，权重值修改后将同步到实例下的所有 ServiceName')"
    />
    <Form
      ref="formRef"
      class="mb-[12px]"
      form-type="vertical"
      :model="formModel"
    >
      <Form.FormItem
        :label="`${$t('权重')}（${$t('数值越大权重越高')}）`"
        property="weight"
        required
        :rules="weightRules"
      >
        <Input
          v-model.trim="formModel.weight"
          :placeholder="$t('请输入 0-10000 的整数')"
          type="number"
        />
      </Form.FormItem>
    </Form>
    <template #footer>
      <Button
        :loading="submitting"
        theme="primary"
        @click="handleConfirm"
      >
        {{ $t('确定') }}
      </Button>
      <Button
        class="ml-[8px]"
        :disabled="submitting"
        @click="handleCancel"
      >
        {{ $t('取消') }}
      </Button>
    </template>
  </Dialog>
</template>

<script lang="ts" setup>
  import { nextTick, reactive, ref, watch } from 'vue';

  import { Alert, Button, Dialog, Form, Input } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { InstanceService } from '~/api/modules/v1';
  import DividerHeader from '~/components/divider-header.vue';
  import { useAppDetail } from '~/stores/app-detail';

  import { runActionSuccess, useInstanceActionContext } from '../../composables/use-instance-action-context';

  const props = defineProps<{
    clearSelectionOnSuccess?: boolean;
    envName: string;
    initialWeight?: null | number;
    instanceIds: string[];
    instanceName: string;
  }>();

  const visible = defineModel<boolean>('visible', { default: false });

  const { t } = useI18n();
  const context = useInstanceActionContext();
  const appDetailStore = useAppDetail();
  const formRef = ref<InstanceType<typeof Form>>();
  const formModel = reactive({ weight: '' });
  const submitting = ref(false);

  const weightRules = [
    {
      required: true,
      message: t('权重不能为空'),
      trigger: 'blur',
    },
    {
      validator: (value: string) => isValidWeight(value),
      message: t('请输入 0-10000 的整数'),
      trigger: 'blur',
    },
  ];

  watch(
    () => visible.value,
    show => {
      if (show) {
        // 多实例批量调整权重时不给默认值，避免误将已有权重统一覆盖为 10
        formModel.weight =
          props.initialWeight === null || props.initialWeight === undefined ? '' : String(props.initialWeight);
        nextTick(() => formRef.value?.clearValidate());
      }
    },
  );

  function handleBeforeClose() {
    return !submitting.value;
  }

  function handleCancel() {
    if (submitting.value) return;
    visible.value = false;
  }

  async function handleConfirm() {
    if (!appDetailStore.appID || !props.envName || !props.instanceIds.length || submitting.value) return;

    // 先加锁再 await 校验：校验期间同样禁止重复提交与关闭弹窗
    submitting.value = true;
    let result = false;
    try {
      const valid = await formRef.value?.validate().catch(() => false);
      if (!valid) return;

      result = await InstanceService.updateAppInstancePolaris({
        appID: appDetailStore.appID,
        envName: props.envName,
        trafficLaneName: '',
        instanceIDs: [...props.instanceIds],
        weight: Number(formModel.weight),
      })
        .then(() => true)
        .catch(() => false);
    } finally {
      submitting.value = false;
    }

    if (!result) return;

    visible.value = false;
    await runActionSuccess(context, { clearSelection: props.clearSelectionOnSuccess });
  }

  // 权重仅允许 0-10000 的整数
  function isValidWeight(value: number | string | undefined) {
    if (value === '' || value === undefined || value === null) return false;
    const weight = Number(value);
    return Number.isInteger(weight) && weight >= 0 && weight <= 10000;
  }
</script>
