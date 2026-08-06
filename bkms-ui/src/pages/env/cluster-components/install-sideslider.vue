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
      <DividerHeader
        :title="props.isUpdate ? $t('更新组件') : $t('安装组件')"
        :title-size="16"
      >
        <div>
          <span>{{ props.addonInfo?.displayName || props.addonInfo?.name }}</span>
          <Tag
            v-if="props.addonInfo?.requiredForAppTypes?.length"
            class="ml-[10px]"
          >
            {{ $t('必选') }}
          </Tag>
        </div>
      </DividerHeader>
    </template>
    <div class="px-[24px] pt-[18px]">
      <!-- 组件配置 -->
      <ComponentsConfig
        ref="configRef"
        v-model:form-data="formData"
        :addon-info="props.addonInfo"
        :components-name="props.componentsName"
        :is-update="props.isUpdate"
      />
    </div>

    <template #footer>
      <div class="flex items-center">
        <Button
          class="mr-[8px]"
          :loading="props.loading"
          theme="primary"
          @click="handleConfirm"
        >
          {{ props.isUpdate ? $t('确认更新') : $t('确认安装') }}
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
  import { computed, ref, watch } from 'vue';

  import { Button, InfoBox, Sideslider, Tag } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { ClusterAddonInfoOutput } from '~/@types/v1/cluster-addon';

  import ComponentsConfig from './components-config.vue';

  interface Emits {
    (e: 'update:visible', value: boolean): void;
    (e: 'cancel'): void;
    (e: 'confirm', data: Record<string, unknown>): void;
  }

  interface Props {
    addonInfo?: ClusterAddonInfoOutput | null;
    componentsName?: string;
    isUpdate?: boolean;
    loading?: boolean;
    visible: boolean;
  }

  const props = withDefaults(defineProps<Props>(), {
    addonInfo: null,
    componentsName: '',
    isUpdate: false,
    loading: false,
  });

  const emit = defineEmits<Emits>();
  const { t } = useI18n();

  const visible = computed({
    get: () => props.visible,
    set: value => emit('update:visible', value),
  });

  // 仅用于调用 validate 方法
  const configRef = ref<InstanceType<typeof ComponentsConfig>>();
  const formData = ref<null | Record<string, unknown>>(null);

  // 表单是否有改动
  const isDirty = ref(false);
  // 初始化标记，防止初始化时触发 dirty
  const initializing = ref(false);
  // 缓存初始表单快照，用于对比
  let initialFormSnapshot = '';

  // 监听 formData 变化进行 dirty 检测
  watch(
    formData,
    newVal => {
      if (initializing.value || !newVal) return;
      const currentSnapshot = JSON.stringify(newVal);
      isDirty.value = currentSnapshot !== initialFormSnapshot;
    },
    { deep: true },
  );

  /**
   * 离开确认弹窗
   * @returns Promise<boolean> true 表示确认离开，false 表示取消
   */
  function confirmBox(): Promise<boolean> {
    return new Promise<boolean>(resolve => {
      if (!isDirty.value) {
        resolve(true);
        return;
      }

      InfoBox({
        title: `${t('确认离开当前页')}？`,
        extCls: 'leave-confirm-box-index',
        content: t('离开将会导致未保存信息丢失'),
        confirmText: t('离开'),
        cancelText: t('取消'),
        onCancel: () => resolve(false),
        onConfirm: () => {
          isDirty.value = false;
          resolve(true);
        },
      });
    });
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
    if (!configRef.value?.validate) return;

    const valid = await configRef.value.validate().catch(() => false);
    if (!valid) return;

    if (!formData.value) return;

    isDirty.value = false;
    emit('confirm', { ...formData.value });
  }

  function handleHidden() {
    // 侧栏关闭时重置状态
    isDirty.value = false;
    initialFormSnapshot = '';
    formData.value = null;
  }

  function handleShown() {
    // 打开时记录初始表单快照并清除 dirty 标记
    initializing.value = true;
    setTimeout(() => {
      initialFormSnapshot = formData.value ? JSON.stringify(formData.value) : '';
      isDirty.value = false;
      initializing.value = false;
    }, 0);
  }
</script>
