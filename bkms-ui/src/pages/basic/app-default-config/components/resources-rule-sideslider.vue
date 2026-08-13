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
        {{ $t('资源规格') }}
      </DividerHeader>
    </template>

    <Form
      ref="formRef"
      class="px-[24px] pt-[24px]"
      form-type="vertical"
      :model="formModel"
      :rules="rules"
    >
      <!-- 资源规格：实例数、CPU、内存 -->
      <div>
        <div class="section-title">{{ $t('资源规格') }}</div>
        <div class="pt-[16px]">
          <Form.FormItem
            :label="$t('实例数')"
            property="replicas"
            required
          >
            <Input
              v-model.trim="formModel.replicas"
              :min="1"
              :precision="0"
              type="number"
            />
          </Form.FormItem>
          <!-- CPU：Requests / Limits 并排 -->
          <div class="flex gap-x-[16px]">
            <Form.FormItem
              class="flex-1"
              :label="`CPU ${$t('预留')} (Requests)`"
              property="cpuRequests"
              required
            >
              <Select
                v-model="formModel.cpuRequests"
                :placeholder="$t('请选择')"
                @change="revalidateRelatedField('cpuLimits')"
              >
                <Select.Option
                  v-for="option in CPU_OPTIONS"
                  :key="option"
                  :label="$t('{0} 核', [getCoreCount(option)])"
                  :value="option"
                />
              </Select>
            </Form.FormItem>
            <Form.FormItem
              class="flex-1"
              :label="`CPU ${$t('限制')} (Limits)`"
              property="cpuLimits"
              required
            >
              <Select
                v-model="formModel.cpuLimits"
                :placeholder="$t('请选择')"
                @change="revalidateRelatedField('cpuRequests')"
              >
                <Select.Option
                  v-for="option in CPU_OPTIONS"
                  :key="option"
                  :label="$t('{0} 核', [getCoreCount(option)])"
                  :value="option"
                />
              </Select>
            </Form.FormItem>
          </div>
          <!-- 内存：Requests / Limits 并排 -->
          <div class="flex gap-x-[16px]">
            <Form.FormItem
              class="flex-1"
              :label="`${$t('内存预留')} (Requests)`"
              property="memoryRequests"
              required
            >
              <Select
                v-model="formModel.memoryRequests"
                :placeholder="$t('请选择')"
                @change="revalidateRelatedField('memoryLimits')"
              >
                <Select.Option
                  v-for="option in MEMORY_OPTIONS"
                  :key="option"
                  :label="option"
                  :value="option"
                />
              </Select>
            </Form.FormItem>
            <Form.FormItem
              class="flex-1"
              :label="`${$t('内存限制')} (Limits)`"
              property="memoryLimits"
              required
            >
              <Select
                v-model="formModel.memoryLimits"
                :placeholder="$t('请选择')"
                @change="revalidateRelatedField('memoryRequests')"
              >
                <Select.Option
                  v-for="option in MEMORY_OPTIONS"
                  :key="option"
                  :label="option"
                  :value="option"
                />
              </Select>
            </Form.FormItem>
          </div>
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
            :supported-env-types="ALL_ENV_TYPES"
          />
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

  import { Button, Form, Input, Message, Select, Sideslider } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';
  import { AppSpecService } from '~/api/modules/v1';
  import DividerHeader from '~/components/divider-header.vue';
  import { useSpaceStore } from '~/stores/space';

  import { ALL_ENV_TYPES, getScopeSubmitEnvTypes } from './app-spec-scope';
  import AppSpecScopeSection from './app-spec-scope-section.vue';

  import type { AppSpecScopeType, AppSpecSliderScene } from './app-spec-scope';
  import type { ResourcesRuleOutputObj, ResourcesSpecInput } from '~/@types/v1/app-spec';

  /** 与应用配置资源规格表单保持一致的可选值 */
  const CPU_OPTIONS = ['0.1', '0.2', '0.5', '1', '2', '4', '8', '16', '32'];
  const MEMORY_OPTIONS = ['256Mi', '512Mi', '1Gi', '2Gi', '4Gi', '8Gi', '16Gi', '32Gi', '64Gi'];

  /** 新建配置项 / 新增规则时的资源规格默认值 */
  const SPEC_DEFAULTS = {
    replicas: 1 as number | string,
    cpuRequests: '1',
    cpuLimits: '2',
    memoryRequests: '2Gi',
    memoryLimits: '4Gi',
  };

  interface Props {
    /** 已存在规则占用的环境类型 */
    occupiedEnvTypes?: string[];
    /** 当前编辑的规则，为空表示新建 */
    rule?: null | ResourcesRuleOutputObj;
    /** 打开场景，用于区分侧滑标题与适用范围选项 */
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

  /** 标题左段：新增配置项 / 新增规则 / 编辑规则 */
  const headerTitle = computed(() => {
    if (props.scene === 'editRule') return t('编辑规则');
    if (props.scene === 'createRule') return t('新增规则');
    return t('新增配置项');
  });

  const formModel = reactive({
    ...SPEC_DEFAULTS,
    scopeType: 'env_type' as AppSpecScopeType,
    envTypes: [] as string[],
  });

  /** CPU 值转核数（用于下拉展示） */
  const getCoreCount = (value: string): number => parseFloat(value);

  /** 内存值（如 '256Mi', '2Gi'）转 MiB，用于 Limits >= Requests 校验 */
  function parseMemoryToMiB(value: string) {
    const match = value.match(/^(\d+(?:\.\d+)?)(Mi|Gi)$/);
    if (!match) return 0;
    const num = parseFloat(match[1]);
    return match[2] === 'Gi' ? num * 1024 : num;
  }

  const scopeParams = computed(() => ({
    scopeType: formModel.scopeType,
    envTypes: formModel.envTypes,
    supportedEnvTypes: ALL_ENV_TYPES,
    occupiedEnvTypes: props.occupiedEnvTypes,
  }));

  /** CPU Limits 不得小于 Requests；规则挂在两侧，避免只改一端时对端不触发 */
  const cpuRelationRule = {
    validator: () => parseFloat(formModel.cpuLimits) >= parseFloat(formModel.cpuRequests),
    message: t('{0} Limits 不能小于 Requests', [t('CPU')]),
    trigger: 'change',
  };
  /** 内存 Limits 不得小于 Requests */
  const memoryRelationRule = {
    validator: () => parseMemoryToMiB(formModel.memoryLimits) >= parseMemoryToMiB(formModel.memoryRequests),
    message: t('{0} Limits 不能小于 Requests', [t('内存')]),
    trigger: 'change',
  };

  const rules = {
    replicas: [
      {
        validator: (value: number | string) => Number(value) >= 1,
        message: t('实例数不能小于1'),
        trigger: 'blur',
      },
    ],
    cpuRequests: [
      {
        validator: () => formModel.cpuRequests.length > 0,
        message: `CPU Requests${t('不能为空')}`,
        trigger: 'change',
      },
      cpuRelationRule,
    ],
    cpuLimits: [
      {
        validator: () => formModel.cpuLimits.length > 0,
        message: `CPU Limits${t('不能为空')}`,
        trigger: 'change',
      },
      cpuRelationRule,
    ],
    memoryRequests: [
      {
        validator: () => formModel.memoryRequests.length > 0,
        message: `${t('内存')} Requests${t('不能为空')}`,
        trigger: 'change',
      },
      memoryRelationRule,
    ],
    memoryLimits: [
      {
        validator: () => formModel.memoryLimits.length > 0,
        message: `${t('内存')} Limits${t('不能为空')}`,
        trigger: 'change',
      },
      memoryRelationRule,
    ],
    envTypes: [
      {
        validator: () => getScopeSubmitEnvTypes(scopeParams.value).length > 0,
        message: t('请选择环境类型'),
        trigger: 'change',
      },
    ],
  };

  /** 组装接口所需的资源规格 */
  function buildSpec(): ResourcesSpecInput {
    return {
      replicas: Number(formModel.replicas),
      cpuRequests: formModel.cpuRequests,
      cpuLimits: formModel.cpuLimits,
      memoryRequests: formModel.memoryRequests,
      memoryLimits: formModel.memoryLimits,
    };
  }

  /** 重置/回填表单：有规则则回填，缺省字段用默认值 */
  function fillForm(rule?: null | ResourcesRuleOutputObj) {
    const spec = rule?.spec;
    Object.assign(formModel, {
      replicas: spec?.replicas ?? SPEC_DEFAULTS.replicas,
      cpuRequests: spec?.cpuRequests || SPEC_DEFAULTS.cpuRequests,
      cpuLimits: spec?.cpuLimits || SPEC_DEFAULTS.cpuLimits,
      memoryRequests: spec?.memoryRequests || SPEC_DEFAULTS.memoryRequests,
      memoryLimits: spec?.memoryLimits || SPEC_DEFAULTS.memoryLimits,
      scopeType: 'env_type',
      envTypes: getScopeSubmitEnvTypes({
        scopeType: 'env_type',
        envTypes: rule?.envTypes ?? [],
        supportedEnvTypes: ALL_ENV_TYPES,
        occupiedEnvTypes: props.occupiedEnvTypes,
      }),
    });
  }

  function handleClosed() {
    formRef.value?.clearValidate?.();
  }

  async function handleConfirm() {
    const valid = await formRef.value?.validate();
    if (!valid || !spaceStore.currentSpace) return;

    saving.value = true;
    try {
      const spec = buildSpec();
      const envTypes = getScopeSubmitEnvTypes(scopeParams.value);
      if (isEdit.value && props.rule?.id) {
        await AppSpecService.updateWorkspaceAppSpecResourcesRule({
          workspaceID: spaceStore.currentSpace,
          ruleID: props.rule.id,
          envTypes,
          spec,
        });
      } else {
        await AppSpecService.createWorkspaceAppSpecResourcesRule({
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

  /**
   * bkui Form 按字段 trigger 校验，改 Requests 不会自动重跑 Limits。
   * 对端变更时手动 validate（bkui 无 validateField）。
   */
  function revalidateRelatedField(property: 'cpuLimits' | 'cpuRequests' | 'memoryLimits' | 'memoryRequests') {
    formRef.value?.validate(property).catch(() => false);
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
</style>
