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
  <BkmsContent
    :collapsible="true"
    :edit-disabled="!currentEnv?.isDefault"
    :edit-disabled-tips="$t('全局配置跨环境共享，请先将环境视角切换到「默认配置」后再修改')"
    :editing="isEditing"
    :show-edit-icon="!isEditing"
    @edit="handleEdit"
  >
    <template #title>
      <div class="flex items-center">
        <div class="text-[14px]">{{ $t('启动配置') }}</div>
        <Tag
          class="ml-[6px] font-400 text-[10px] h-[20px] leading-[20px]"
          type="stroke"
        >
          {{ $t('全局') }}
        </Tag>
      </div>
    </template>
    <div class="bg-[#FFF] p-[16px]">
      <!-- 查看态 -->
      <template v-if="!isEditing">
        <FieldItem
          class="mb-[12px]"
          :field-value="$t('启动模式')"
          :field-width="205"
          :value="appModelSpec?.shCommand ? 'shell' : 'exec'"
        />
        <FieldItem
          v-if="appModelSpec?.shCommand"
          class="!items-start"
          container-height="auto"
          :field-value="$t('启动命令')"
          :field-width="205"
        >
          <template #value>
            {{ appModelSpec.shCommand }}
          </template>
        </FieldItem>
        <div
          v-else
          class="grid grid-cols-2 gap-[12px] gap-y-2"
        >
          <FieldItem
            class="min-h-[20px]"
            :class="[appModelSpec?.command?.length ? '!items-start' : '']"
            container-height="auto"
            :field-value="$t('启动命令')"
            :field-width="205"
          >
            <template #value>
              <div class="flex flex-col gap-[4px] w-full min-w-0">
                <div
                  v-for="item in appModelSpec?.command || []"
                  :key="item"
                >
                  <OverflowTitle type="tips">{{ item }}</OverflowTitle>
                </div>
                <span
                  v-if="!appModelSpec?.command?.length"
                  class="text-[12px] text-[#313238]"
                  >--</span
                >
              </div>
            </template>
          </FieldItem>
          <FieldItem
            class="min-h-[20px]"
            :class="[appModelSpec?.args?.length ? '!items-start' : '']"
            container-height="auto"
            :field-value="$t('命令参数')"
            :field-width="205"
          >
            <template #value>
              <div class="flex flex-col gap-[4px] w-full min-w-0">
                <div
                  v-for="item in appModelSpec?.args || []"
                  :key="item"
                  class="w-full overflow-hidden"
                >
                  <OverflowTitle type="tips">{{ item }}</OverflowTitle>
                </div>
                <span
                  v-if="!appModelSpec?.args?.length"
                  class="text-[12px] text-[#313238]"
                  >--
                </span>
              </div>
            </template>
          </FieldItem>
        </div>
      </template>
      <!-- 编辑态 -->
      <div v-else>
        <StartupCommandInput
          ref="startupCommandRef"
          class="max-w-[1000px] err-relative"
          :value="appModelSpec"
        />

        <div class="!mb-0 mt-[16px] flex items-center">
          <Button
            :loading="saving"
            theme="primary"
            @click="handleSave"
          >
            {{ $t('保存') }}
          </Button>
          <Button
            class="ml-[8px]"
            @click="handleCancel"
          >
            {{ $t('取消') }}
          </Button>
        </div>
      </div>
    </div>
  </BkmsContent>
</template>

<script setup lang="ts">
  import { computed, ref } from 'vue';

  import { Button, Message, OverflowTitle, Tag } from 'bkui-vue';
  import { cloneDeep } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import BkmsContent from '~/components/bkms-content.vue';
  import FieldItem from '~/components/field-item.vue';
  import StartupCommandInput from '~/components/startup-command-input.vue';
  import useSpecField from '~/composables/use-spec-field';
  import { useAppDetail } from '~/stores/app-detail';

  import type { ExtendedEnv } from './types';
  import type { AppModelSpecInput } from '~/@types/v1/app';

  interface Props {
    currentEnv: ExtendedEnv | null;
  }

  const props = defineProps<Props>();

  const { t } = useI18n();
  const appDetailStore = useAppDetail();
  const { updateSpecApi } = useSpecField();

  // 从 store 获取 appModelSpec
  const appModelSpec = computed(() => appDetailStore.appDetail?.appModelSpec);

  // ========== 编辑态控制 ==========
  const isEditing = ref(false);
  const saving = ref(false);
  const startupCommandRef = ref<InstanceType<typeof StartupCommandInput> | null>(null);

  /** 取消编辑 */
  function handleCancel() {
    isEditing.value = false;
  }

  /** 进入编辑态 */
  function handleEdit() {
    if (!props.currentEnv?.isDefault) return;
    isEditing.value = true;
  }

  /** 保存编辑 */
  async function handleSave() {
    const appDetail = appDetailStore.appDetail;
    if (!appDetail || !props.currentEnv?.isDefault || !startupCommandRef.value || saving.value) return;
    if (!(await startupCommandRef.value.validate())) return;

    saving.value = true;
    const updatedAppModelSpec = cloneDeep(appDetail.appModelSpec || {}) as AppModelSpecInput;
    Object.assign(updatedAppModelSpec, startupCommandRef.value.getValue());

    try {
      const result = await updateSpecApi
        .value({
          appID: appDetailStore.appID,
          appModelSpec: updatedAppModelSpec,
        })
        .then(() => true)
        .catch(() => false);

      if (result) {
        Message({
          message: t('操作成功'),
          theme: 'success',
        });
        await appDetailStore.fetchAppDetail();
        isEditing.value = false;
      }
    } finally {
      saving.value = false;
    }
  }
</script>

<style lang="postcss" scoped>
  :deep(.bkms-content-title) {
    background-color: #eaebf0;
  }
</style>
