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
    :quick-close="false"
    width="480"
    @confirm="handleConfirm"
  >
    <template #header>
      <DividerHeader :title="$t('调整权重')">
        <span class="truncate">{{ instance?.id }}</span>
      </DividerHeader>
    </template>
    <Alert
      class="mb-[16px]"
      closable
      theme="info"
      :title="$t('权重为 0 时 该实例将不再接收流量，权重值修改后将同步到实例下的所有 ServiceName')"
    />
    <Form
      class="mb-[12px]"
      form-type="vertical"
    >
      <Form.FormItem
        :label="`${$t('权重')}（${$t('数值越大权重越高')}）`"
        required
      >
        <Input
          v-model.trim="weightValue"
          :min="0"
          type="number"
        />
      </Form.FormItem>
    </Form>
  </Dialog>
</template>

<script lang="ts" setup>
  import { ref, watch } from 'vue';

  import { Alert, Dialog, Form, Input } from 'bkui-vue';
  import { type AppInstanceOutputObj } from '~/@types/v1/instance';
  import { InstanceService } from '~/api/modules/v1';
  import DividerHeader from '~/components/divider-header.vue';
  import { useAppDetail } from '~/stores/app-detail';

  import { runActionSuccess, useInstanceActionContext } from '../../composables/use-instance-action-context';

  const props = defineProps<{
    envName: string;
    instance: AppInstanceOutputObj | null;
  }>();

  const visible = defineModel<boolean>('visible', { default: false });

  const context = useInstanceActionContext();
  const appDetailStore = useAppDetail();
  const weightValue = ref(0);
  const submitting = ref(false);

  watch(
    () => [visible.value, props.instance] as const,
    ([show, inst]) => {
      if (show && inst) {
        weightValue.value = Number(inst.polarisInfos?.[0]?.weight ?? 10);
      }
    },
  );

  async function handleConfirm() {
    if (!appDetailStore.appID || !props.envName || !props.instance || submitting.value) return;

    submitting.value = true;
    const result = await InstanceService.updateAppInstancePolaris({
      appID: appDetailStore.appID,
      envName: props.envName,
      trafficLaneName: '',
      instanceIDs: [props.instance.id!],
      weight: Number(weightValue.value),
    })
      .then(() => true)
      .catch(() => false);
    submitting.value = false;

    if (!result) return;

    visible.value = false;
    await runActionSuccess(context);
  }
</script>
