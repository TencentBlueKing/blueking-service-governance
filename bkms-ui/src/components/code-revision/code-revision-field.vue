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
  <Form.FormItem
    v-if="showSource"
    :label="t('代码来源')"
  >
    <Radio.Group v-model="typeModel">
      <Radio label="branch">
        {{ t('分支') }}
      </Radio>
      <Radio label="tag"> Tag </Radio>
      <Radio label="commit"> CommitID </Radio>
    </Radio.Group>
  </Form.FormItem>
  <Form.FormItem
    ref="revisionFormItemRef"
    data-testid="code-revision-field"
    :description="description"
    :label="t(fieldLabelKey)"
    :property="property"
    :required="required"
    :rules="fieldRules"
  >
    <!-- CommitID 没有列表，只做格式校验 -->
    <Input
      v-if="showSource && revisionType === 'commit'"
      ref="commitInputRef"
      v-model.trim="commitModel"
      :class="controlClass"
      clearable
      :disabled="disabled"
    />
    <!-- 按来源重建下拉，避免切到 Tag 后还拿着上一次分支名做过滤，列表被滤空 -->
    <RepoRefSelect
      v-else
      :key="repoGroup || 'mixed'"
      ref="repoRef"
      :class="controlClass"
      :disabled="disabled"
      :group="repoGroup"
      :model-value="modelValue"
      :repository-id="repositoryId"
      :workspace-id="workspaceId"
      @branch-commit="emit('confirm')"
      @update:model-value="emit('update:modelValue', $event)"
    />
  </Form.FormItem>
</template>

<script lang="ts" setup>
  import { computed, nextTick, onBeforeUnmount, ref } from 'vue';

  import { Form, Input, Radio } from 'bkui-vue';
  import { debounce } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import RepoRefSelect from '~/components/repo-ref-select/repo-ref-select.vue';

  import { type RevisionType, isCommitID, revisionFieldLabel } from './revision';

  const props = withDefaults(
    defineProps<{
      /** 输入框宽度等样式，由各入口自己决定 */
      controlClass?: string;
      /** 切回分支时重新带出的默认修订 */
      defaultRevision?: string;
      description?: string;
      disabled?: boolean;
      modelValue?: string;
      property?: string;
      repositoryId?: string;
      required?: boolean;
      revisionType?: RevisionType;
      /** 自定义流水线不展示来源单选，第二项保持原来的代码分支 */
      showSource?: boolean;
      workspaceId: string;
    }>(),
    {
      controlClass: '',
      defaultRevision: '',
      description: '',
      disabled: false,
      modelValue: '',
      property: 'revision',
      repositoryId: '',
      required: false,
      revisionType: 'branch',
      showSource: true,
    },
  );

  const emit = defineEmits<{
    /** 修订已确认，父级用来重算推荐版本号 */
    confirm: [];
    'update:modelValue': [value: string];
    'update:revisionType': [value: RevisionType];
  }>();

  const { t } = useI18n();
  const repoRef = ref<InstanceType<typeof RepoRefSelect>>();
  const commitInputRef = ref<InstanceType<typeof Input>>();
  /** 修订表单项：来源切换后清掉带着旧字段名的错误，例如切回分支仍显示「代码 Tag不能为空」 */
  const revisionFormItemRef = ref<{ clearValidate: () => void }>();

  const fieldLabelKey = computed(() => (props.showSource ? revisionFieldLabel(props.revisionType) : '代码分支'));
  // 流水线模式不收窄列表，仍是原来的分支 / Tag 混合下拉
  const repoGroup = computed(() => {
    if (!props.showSource || props.revisionType === 'commit') return undefined;
    return props.revisionType;
  });

  const fieldRules = computed(() => [
    {
      // 与原先执行构建的表单规则一致：不展示必填星号时，空修订仍然不能提交
      validator: (value: string) => Boolean(value?.trim()),
      message: t('不能为空'),
      trigger: ['blur', 'change'],
    },
    {
      validator: (value: string) => {
        if (!props.showSource || props.revisionType !== 'commit' || !value?.trim()) return true;
        return isCommitID(value);
      },
      message: t('格式错误'),
      trigger: ['blur', 'change'],
    },
  ]);

  const typeModel = computed({
    get: () => props.revisionType,
    set: (value: RevisionType) => {
      handleTypeChange(value);
    },
  });

  /**
   * 停止输入约 0.5 秒后再确认。
   * 不合法的 CommitID 也要通知父级，用来作废还在飞的推荐请求，版本号本身不会被重算。
   */
  const debouncedCommitConfirm = debounce(() => {
    emit('confirm');
  }, 500);

  // 模板使用 v-model.trim。bkui Input 不读取 modelModifiers，首尾空白在这里再去掉一次
  const commitModel = computed({
    get: () => props.modelValue,
    set: (value: string) => {
      const next = value.trim();
      emit('update:modelValue', next);

      // 清空后立刻按「不含修订」重算，不等 0.5 秒
      if (!next) {
        debouncedCommitConfirm.cancel();
        emit('confirm');
        return;
      }

      debouncedCommitConfirm();
    },
  });

  onBeforeUnmount(() => {
    debouncedCommitConfirm.cancel();
  });

  function containsTarget(target: Node | null) {
    if (props.showSource && props.revisionType === 'commit') {
      const inputEl = (commitInputRef.value as undefined | { $el?: HTMLElement })?.$el;
      return inputEl?.contains(target) ?? false;
    }
    return repoRef.value?.containsTarget(target) ?? false;
  }

  function dismissIfOutside(target: Node | null) {
    repoRef.value?.dismissIfOutside(target);
  }

  /**
   * 切到 Tag / CommitID 时清空，避免把上一个分支名带过去。
   * 切回分支时重新带出默认修订，并清掉切换过程中失焦留下的旧校验。
   */
  function handleTypeChange(type: RevisionType) {
    if (type === props.revisionType) return;

    debouncedCommitConfirm.cancel();
    const next = type === 'branch' ? props.defaultRevision : '';
    emit('update:revisionType', type);
    emit('update:modelValue', next);

    // 单选会让下拉先失焦，空值校验发生在来源改完之前，错误文案会停在旧字段名上
    nextTick(() => {
      if (type !== 'commit') {
        repoRef.value?.prepare(next);
      }
      revisionFormItemRef.value?.clearValidate?.();
    });

    emit('confirm');
  }

  function hidePopover() {
    repoRef.value?.hidePopover();
  }

  /** 打开表单时回填默认分支；CommitID 输入框没有预拉列表 */
  function prepare(preferred = '') {
    if (repoRef.value) {
      return repoRef.value.prepare(preferred);
    }
    emit('update:modelValue', preferred);
    if (preferred.trim()) {
      emit('confirm');
    }
    return preferred;
  }

  function reset() {
    debouncedCommitConfirm.cancel();
    repoRef.value?.reset();
  }

  defineExpose({
    prepare,
    hidePopover,
    reset,
    containsTarget,
    dismissIfOutside,
  });
</script>
