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
    width="640"
    @confirm="handleConfirm"
  >
    <template #header>
      <DividerHeader :title="$t('灰度实例')">
        <span
          v-if="instances.length > 1"
          class="truncate"
        >
          {{ $t('已选 {0} 个实例', [instances.length]) }}
        </span>
        <span
          v-else-if="instances.length === 1"
          class="truncate"
        >
          {{ instances[0]?.id }}
        </span>
        <span
          v-if="envDisplayName"
          class="truncate"
        >
          {{ `${$t('环境')}: ${envDisplayName}` }}
        </span>
      </DividerHeader>
    </template>
    <Form
      class="mt-[4px] mb-[12px]"
      form-type="vertical"
    >
      <Form.FormItem
        :label="$t('镜像 Tag')"
        required
      >
        <ImageSelect v-model:value="imageTag" />
      </Form.FormItem>
    </Form>
  </Dialog>
</template>

<script lang="ts" setup>
  import { ref, watch } from 'vue';

  import { Dialog, Form } from 'bkui-vue';
  import { type AppInstanceOutputObj } from '~/@types/v1/instance';
  import { ApiServerService } from '~/api/modules/bkmsserver';
  import DividerHeader from '~/components/divider-header.vue';
  import ImageSelect from '~/pages/application/components/image-select.vue';
  import { useAppDetail } from '~/stores/app-detail';

  import { runActionSuccess, useInstanceActionContext } from '../../composables/use-instance-action-context';

  const props = defineProps<{
    envDisplayName?: string;
    envName: string;
    /** 跨页全选时传空数组表示全量灰度 */
    instanceIds?: string[];
    instances: AppInstanceOutputObj[];
  }>();

  const visible = defineModel<boolean>('visible', { default: false });

  const emit = defineEmits<{
    success: [];
  }>();

  const context = useInstanceActionContext();
  const appDetailStore = useAppDetail();
  const imageTag = ref('');
  const submitting = ref(false);

  watch(visible, show => {
    if (show) {
      imageTag.value = '';
    }
  });

  async function handleConfirm() {
    if (!appDetailStore.appID || !imageTag.value || !props.envName || submitting.value) return;

    const ids =
      props.instanceIds !== undefined ? props.instanceIds : (props.instances?.map(item => item.id) as string[]);

    submitting.value = true;
    const result = await ApiServerService.UpdateAppInstances({
      appID: appDetailStore.appID,
      envName: props.envName,
      imageTag: imageTag.value,
      updateStrategy: 'InplaceUpdate',
      instanceIDs: ids,
    })
      .then(() => true)
      .catch(() => false);
    submitting.value = false;

    if (!result) return;

    visible.value = false;
    await runActionSuccess(context, { clearSelection: true });
    emit('success');
  }
</script>
