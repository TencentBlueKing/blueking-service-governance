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
    v-model:is-show="isShow"
    :width="480"
    @closed="handleClose"
  >
    <template #header>
      <div class="flex flex-col items-center">
        <SvgIcon
          :height="42"
          icon="bkms-icon-tishi"
          :width="42"
        />
        <span class="text-[#313238] text-[20px] leading-[32px] text-center mt-[18px]">
          {{ $t('确定要删除公共变量') }}?
        </span>
      </div>
    </template>

    <!-- 变量信息 -->
    <div class="bg-[#F5F7FA] mb-[16px] py-[12px] px-[16px] text-[12px]">
      <ul class="list-disc line-height-[20px] font-normal">
        <li>
          <span class="text-[#979BA5]">{{ $t('变量名') }}：</span>
          <span class="text-[#313238]">{{ envVarData?.key }}</span>
        </li>
        <li>
          <span class="text-[#979BA5]">{{ $t('生效环境类型') }}：</span>
          <span class="text-[#313238]">{{ scopeLabel }}</span>
        </li>
      </ul>
    </div>

    <Checkbox v-model="confirmed">
      {{ $t('我已了解风险，确认删除') }}
    </Checkbox>

    <template #footer>
      <div class="flex justify-center">
        <span
          v-bk-tooltips="{
            content: $t('请先勾选确认'),
            disabled: confirmed,
          }"
        >
          <Button
            class="mr-[8px]"
            :disabled="!confirmed"
            :loading="deleting"
            theme="danger"
            @click="handleDelete"
          >
            {{ $t('删除') }}
          </Button>
        </span>
        <Button @click="handleClose">
          {{ $t('取消') }}
        </Button>
      </div>
    </template>
  </Dialog>
</template>

<script lang="ts" setup>
  import { computed, ref } from 'vue';

  import { Button, Checkbox, Dialog, Message } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { ScopedEnvVarOutputObj } from '~/@types/v1/envvars';
  import { EnvvarsService } from '~/api/modules/v1/envvars';
  import { getScopeLabel } from '~/composables/use-scope-display';

  interface Props {
    envVarData?: null | ScopedEnvVarOutputObj;
    workspaceId: string;
  }

  const props = withDefaults(defineProps<Props>(), {
    envVarData: null,
  });

  const emit = defineEmits<{ deleted: [] }>();

  const { t } = useI18n();

  const isShow = defineModel<boolean>('isShow');

  const confirmed = ref(false);
  const deleting = ref(false);

  /** 作用域展示标签 */
  const scopeLabel = computed(() => {
    if (props.envVarData && props.envVarData?.scopeType && props.envVarData?.scopeValue) {
      return getScopeLabel(props.envVarData.scopeType, props.envVarData.scopeValue);
    }
    return t('所有环境');
  });

  function handleClose() {
    isShow.value = false;
    confirmed.value = false;
  }

  async function handleDelete() {
    if (!props.envVarData) return;

    try {
      deleting.value = true;
      await EnvvarsService.deleteScopedEnvVar(
        { workspaceID: props.workspaceId, scopedEnvVarID: props.envVarData?.id || '' },
        { validateCode: false },
      );
      Message({ message: t('删除成功'), theme: 'success' });
      handleClose();
      emit('deleted');
    } catch (err) {
      console.error(err);
    } finally {
      deleting.value = false;
    }
  }
</script>

<style lang="postcss" scoped>
  :deep(.bk-dialog-header) {
    padding-top: 48px;
  }

  :deep(.bk-dialog-footer) {
    border: none;
    background-color: unset;
    padding-top: 0;
    padding-bottom: 24px;
  }
</style>
