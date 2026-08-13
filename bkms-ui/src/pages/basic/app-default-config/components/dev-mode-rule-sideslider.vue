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
    quick-close
    render-directive="if"
    :width="928"
    @closed="handleClosed"
  >
    <template #header>
      <DividerHeader>
        <template #title>{{ headerTitle }}</template>
        {{ $t('开发模式') }}
      </DividerHeader>
    </template>

    <Form
      ref="formRef"
      class="px-[24px] pt-[24px]"
      form-type="vertical"
      :model="formModel"
      :rules="rules"
    >
      <!-- 开发模式：默认开启开关 -->
      <div>
        <div class="section-title">{{ $t('开发模式') }}</div>
        <div class="pt-[16px]">
          <Form.FormItem
            class="section-mode_label"
            :label="$t('默认开启开发模式')"
            property="enabled"
          >
            <div class="flex items-center text-[#4D4F56]">
              <Switcher
                v-model="formModel.enabled"
                class="mr-[6px]"
                theme="primary"
              />
              {{ $t('支持通过 bkms-cli 上传二进制的方式热更新服务，更新过程不会重启容器实例。') }}
              <Button
                text
                theme="primary"
                @click="goToTrpcDevModeDoc"
              >
                {{ $t('查看详细文档') }}
                <Share class="ml-[6px]" />
              </Button>
            </div>
          </Form.FormItem>
        </div>
      </div>

      <!-- 适用范围 -->
      <div>
        <div class="section-title">
          {{ $t('适用范围') }}
          <i
            v-bk-tooltips="{
              content: $t('同一适用范围在本配置项中只能有一条规则；越具体的范围优先级越高。'),
              placement: 'top',
            }"
            class="bkms-icon bkms-icon-warning-circle ml-[8px] text-[16px] text-[#979BA5]"
          ></i>
        </div>
        <div class="pt-[16px]">
          <AppSpecScopeSection
            v-model:env-types="formModel.envTypes"
            v-model:scope-type="formModel.scopeType"
            :occupied-env-types="occupiedEnvTypes"
            :supported-env-types="DEV_MODE_SUPPORTED_ENV_TYPES"
          />
          <!-- 开发模式侧滑暂不支持「所有环境」
          :show-all-env-option="showAllEnvOption"
          -->
        </div>
      </div>
    </Form>

    <template #footer>
      <div class="footer-actions">
        <Button
          class="!w-[60px] !min-w-[60px]"
          :loading="saving"
          theme="primary"
          @click="handleConfirm"
        >
          {{ $t('确定') }}
        </Button>
        <Button
          class="!w-[60px] !min-w-[60px]"
          @click="visible = false"
        >
          {{ $t('取消') }}
        </Button>
      </div>
    </template>
  </Sideslider>
</template>

<script lang="ts" setup>
  import { computed, reactive, ref, watch } from 'vue';

  import { Button, Form, Message, Sideslider, Switcher } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { AppSpecService } from '~/api/modules/v1';
  import { DOC_LINKS } from '~/common/const';
  import DividerHeader from '~/components/divider-header.vue';
  import { useSpaceStore } from '~/stores/space';

  import { DEV_MODE_SUPPORTED_ENV_TYPES, getScopeSubmitEnvTypes } from './app-spec-scope';
  import AppSpecScopeSection from './app-spec-scope-section.vue';

  import type { AppSpecScopeType, AppSpecSliderScene } from './app-spec-scope';
  import type { DevModeRuleOutputObj } from '~/@types/v1/app-spec';

  interface Props {
    /** 已存在规则占用的环境类型 */
    occupiedEnvTypes?: string[];
    /** 当前编辑的规则，为空表示新建 */
    rule?: DevModeRuleOutputObj | null;
    /** 打开场景，用于区分侧滑标题 */
    scene?: AppSpecSliderScene;
  }

  const props = withDefaults(defineProps<Props>(), {
    occupiedEnvTypes: () => [],
    rule: null,
    scene: 'addConfig',
  });

  const emit = defineEmits<{
    success: [];
  }>();

  const visible = defineModel<boolean>('isShow', { default: false });

  const { t } = useI18n();
  const spaceStore = useSpaceStore();
  const formRef = ref<InstanceType<typeof Form>>();
  const saving = ref(false);

  const isEdit = computed(() => props.scene === 'editRule' && Boolean(props.rule?.id));

  /** 开发模式侧滑暂不支持「所有环境」
  const showAllEnvOption = computed(() => props.scene === 'addConfig');
  */

  /** 标题左段：新增配置项 / 新增规则 / 编辑规则 */
  const headerTitle = computed(() => {
    if (props.scene === 'editRule') return t('编辑规则');
    if (props.scene === 'createRule') return t('新增规则');
    return t('新增配置项');
  });

  const formModel = reactive({
    enabled: true,
    scopeType: 'env_type' as AppSpecScopeType,
    envTypes: [] as string[],
  });

  const scopeParams = computed(() => ({
    scopeType: formModel.scopeType,
    envTypes: formModel.envTypes,
    supportedEnvTypes: DEV_MODE_SUPPORTED_ENV_TYPES,
    occupiedEnvTypes: props.occupiedEnvTypes,
  }));

  const rules = {
    envTypes: [
      {
        validator: () => getScopeSubmitEnvTypes(scopeParams.value).length > 0,
        message: t('请选择环境类型'),
        trigger: 'change',
      },
    ],
  };

  /** 重置/回填表单 */
  function fillForm(rule?: DevModeRuleOutputObj | null) {
    formModel.enabled = rule ? Boolean(rule.spec?.enabled) : true;
    formModel.scopeType = 'env_type';
    formModel.envTypes = getScopeSubmitEnvTypes({
      scopeType: 'env_type',
      envTypes: rule?.envTypes ?? [],
      supportedEnvTypes: DEV_MODE_SUPPORTED_ENV_TYPES,
      occupiedEnvTypes: props.occupiedEnvTypes,
    });
  }

  function goToTrpcDevModeDoc() {
    window.open(`${import.meta.env.BK_DOC_URL}${DOC_LINKS.TRPC_DEV_MODE}`, '_blank');
  }

  function handleClosed() {
    formRef.value?.clearValidate?.();
  }

  async function handleConfirm() {
    const valid = await formRef.value?.validate();
    if (!valid || !spaceStore.currentSpace) return;

    saving.value = true;
    try {
      const spec = { enabled: formModel.enabled };
      const envTypes = getScopeSubmitEnvTypes(scopeParams.value);
      if (isEdit.value && props.rule?.id) {
        await AppSpecService.updateWorkspaceAppSpecDevModeRule({
          workspaceID: spaceStore.currentSpace,
          ruleID: props.rule.id,
          envTypes,
          spec,
        });
      } else {
        await AppSpecService.createWorkspaceAppSpecDevModeRule({
          workspaceID: spaceStore.currentSpace,
          envTypes,
          spec,
        });
      }
      Message({
        theme: 'success',
        message: props.scene === 'addConfig' ? t('已保存，将应用于后续创建/部署') : t('操作成功'),
      });
      visible.value = false;
      emit('success');
    } finally {
      saving.value = false;
    }
  }

  watch(visible, isShow => {
    if (isShow) {
      fillForm(props.rule);
    }
  });
</script>

<style lang="postcss" scoped>
  :deep(.bk-sideslider-footer) {
    margin-top: 8px;
  }

  .footer-actions {
    display: flex;
    gap: 6px;
  }

  .section-title {
    display: flex;
    align-items: center;
    height: 32px;
    padding: 0 16px;
    color: #313238;
    font-size: 14px;
    font-weight: 700;
    line-height: 22px;
    background-color: #f5f7fa;
    border-radius: 2px;
  }

  :deep(.section-mode_label .bk-form-label) {
    color: #4d4f56;
    font-size: 12px;
  }
</style>
