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
    v-model:is-show="isShow"
    quick-close
    render-directive="if"
    :width="960"
  >
    <template #header>
      <span class="align-bottom">
        <span>{{ t('组件详情') }}</span>
        <span class="ml-[12px] mr-[9px] text-[#DCDEE5]">|</span>
        <span class="text-[14px] text-[#979BA5]">
          {{ row.displayName || row.name || '--' }}
        </span>
      </span>
    </template>
    <template #default>
      <div
        v-bkloading="{ loading: isLoading }"
        class="h-[calc(100vh-52px)] flex flex-col overflow-hidden"
      >
        <div
          v-bkloading="{ loading: isUpdateLoading }"
          class="w-full px-[24px] py-[18px] bg-[#F5F7FA] border-[1px] border-solid border-[#F0F1F5]"
        >
          <ComponentLabel :label="langMap.label.baseInfo">
            <div class="grid grid-cols-3 px-[20px]">
              <DetailItem :label="langMap.label.ID">
                <div>{{ row.name || '--' }}</div>
              </DetailItem>
              <DetailItem :label="langMap.label.name">
                <EditBlock
                  v-if="allowedRange"
                  class="absolute"
                  :is-show-at-first-line="true"
                  @cancel="init"
                  @confirm="handleConfirm"
                >
                  <template #text>
                    <div>{{ formData.displayName }}</div>
                  </template>
                  <template #edit="{ focus }">
                    <Input
                      :ref="(el: InputType) => focus(el, 'input')"
                      v-model="formData.displayName"
                      autosize
                      class="mr-[8px] max-w-[120px] min-h-[30px]"
                      :resize="false"
                      type="textarea"
                    />
                  </template>
                </EditBlock>
                <div v-else>{{ formData.displayName }}</div>
              </DetailItem>
              <DetailItem :label="langMap.label.currentVersion">
                <div>{{ row.version || '--' }}</div>
              </DetailItem>
              <DetailItem :label="langMap.label.type">
                <div>{{ row.type || '--' }}</div>
              </DetailItem>
              <DetailItem :label="langMap.label.openScope">
                <div>{{ row.type || '--' }}</div>
              </DetailItem>
              <DetailItem
                class="relative w-full"
                :label="langMap.label.description"
              >
                <EditBlock
                  v-if="allowedRange"
                  class="absolute"
                  :is-show-at-first-line="true"
                  @cancel="init"
                  @confirm="handleConfirm"
                >
                  <template #text>
                    <div>{{ formData.description }}</div>
                  </template>
                  <template #edit="{ focus }">
                    <Input
                      :ref="(el: InputType) => focus(el, 'input')"
                      v-model="formData.description"
                      autosize
                      class="mr-[8px] max-w-[120px] min-h-[30px]"
                      :resize="false"
                      type="textarea"
                    />
                  </template>
                </EditBlock>
                <div v-else>{{ formData.description }}</div>
              </DetailItem>
            </div>
          </ComponentLabel>
        </div>
        <div
          v-bkloading="{ loading: isLoading }"
          class="grow flex overflow-y-hidden"
        >
          <ComponentLabel
            class="scrollbar-stable w-[210px] overflow-y-auto pl-[24px] pr-[18px] py-[16px] bg-[#FAFBFD] border-[1px] border-solid border-[#F0F1F5]"
            :label="langMap.label.componentVersion"
          >
            <div class="flex flex-col gap-[8px]">
              <Button
                v-if="allowedRange"
                class="create-version"
                theme="primary"
                @click="handleCreateVersion"
              >
                <div class="flex items-center justify-center w-full">
                  <Plus
                    height="24"
                    width="24"
                  />
                  {{ langMap.button.createVersion }}
                </div>
              </Button>
              <Input
                v-model.trim="searchValue"
                class="w-full"
                type="search"
              />
              <template
                v-for="item in filterVersionList"
                :key="item.version"
              >
                <FlexRow
                  average
                  :class="[
                    'version-content w-full px-[8px] border-[#DCDEE5]',
                    'border-[1px] rounded-[2px] h-[32px] cursor-pointer',
                    currentVersion.version === item.version ? 'version-active' : '',
                  ]"
                  @click="handleChangeVersion(item)"
                >
                  <template #left>
                    <div class="flex items-center">
                      <div class="max-w-[50px] ellipsis">
                        <span class="version text-[#4D4F56]">{{ item.version }}</span>
                      </div>
                      <Tag
                        v-if="item.version === versionList[0].version"
                        class="!bg-[#CBF0DA] ml-[1px]"
                        radius="8px"
                        size="small"
                        theme="success"
                      >
                        latest
                      </Tag>
                    </div>
                  </template>
                  <template #right>
                    <div class="flex items-center justify-end">
                      <PopConfirm
                        :confirm-config="{
                          theme: 'danger',
                        }"
                        :confirm-text="langMap.button.delete"
                        placement="bottom-start"
                        :popover-options="{
                          boundary: 'document.body',
                        }"
                        :title="langMap.tips.title2"
                        trigger="click"
                        @confirm="handleDeleteVersion(item)"
                      >
                        <Button
                          v-bk-tooltips="{
                            content:
                              versionList.length === 1
                                ? item.referenceCount > 0
                                  ? langMap.tips.notDelete
                                  : langMap.tips.notDelete1
                                : item.referenceCount > 0
                                  ? langMap.tips.notDelete
                                  : langMap.tips.canDelete,
                          }"
                          class="hidden delete-version"
                          :disabled="item.referenceCount > 0 || versionList.length === 1"
                          text
                          theme="primary"
                          @blur="$event.currentTarget.classList.add('hidden')"
                          @click="$event.currentTarget.classList.remove('hidden')"
                        >
                          <Del
                            height="16"
                            width="16"
                          />
                        </Button>
                        <template #content>
                          <div class="flex flex-col mb-[16px]">
                            <span>{{ langMap.label.version3 }}: {{ item.version }}</span>
                            <span>{{ langMap.tips.content2 }}</span>
                          </div>
                        </template>
                      </PopConfirm>
                      <div
                        v-bk-tooltips="$t(`引用数: ${item.referenceCount}`)"
                        class="reference-count ml-[5px] leading-[16px] min-w-[23px] text-center rounded-[8px] bg-[#F0F1F5] text-[12px] text-[#979BA5]"
                      >
                        {{ item.referenceCount }}
                      </div>
                    </div>
                  </template>
                </FlexRow>
              </template>
            </div>
          </ComponentLabel>
          <div class="scrollbar-stable flex-1 overflow-y-auto pl-[24px] pr-[18px] py-[16px] flex flex-col gap-[24px]">
            <ComponentLabel :label="`${currentVersion.version}${langMap.label.version2}`">
              <div class="grid grid-cols-2 px-[20px]">
                <DetailItem
                  class="items-center gap-[8px]"
                  :label="langMap.label.version3"
                >
                  <div class="text-[#313238]">{{ currentVersion.version }}</div>
                </DetailItem>
                <DetailItem
                  class="items-center gap-[8px]"
                  :label="langMap.label.importNumber"
                >
                  <div class="text-[#313238]">{{ currentVersion.referenceCount }}</div>
                </DetailItem>
                <DetailItem
                  class="items-center gap-[8px]"
                  :label="langMap.label.createTime"
                >
                  <div class="text-[#313238]">
                    {{ currentVersion.createTime ? formatDateString(currentVersion.createTime) : '--' }}
                  </div>
                </DetailItem>
                <DetailItem
                  class="items-center gap-[8px]"
                  :label="langMap.label.creator"
                >
                  <div class="text-[#313238]">{{ currentVersion.creator }}</div>
                </DetailItem>
              </div>
            </ComponentLabel>
            <ComponentLabel
              :label="langMap.label.inputParams"
              :show-icon="false"
            >
              <MsEditor
                ref="inputParamsRef"
                class="w-full h-[300px]"
                :model-value="inputParamsValue"
                :readonly="true"
                :title="langMap.label.inputParams1"
              />
            </ComponentLabel>
            <ComponentLabel
              :label="langMap.label.outputParams"
              :show-icon="false"
            >
              <MsEditor
                ref="outputParamsRef"
                class="w-full h-[300px]"
                :model-value="outputParamsValue"
                :readonly="true"
                :title="langMap.label.outputParams1"
              />
            </ComponentLabel>
          </div>
        </div>
      </div>
    </template>
  </Sideslider>
  <CreateVersion
    ref="CreateVersionRef"
    :allowed-range="allowedRange"
    :current-component="row"
    @refresh="getComponentInfo"
  >
  </CreateVersion>
</template>

<script setup lang="ts">
  import { computed, ref, watch } from 'vue';

  import { Button, Input, PopConfirm, Sideslider, Tag } from 'bkui-vue';
  import { Del, Plus } from 'bkui-vue/lib/icon';
  import yaml from 'js-yaml';
  import { cloneDeep, isEmpty } from 'lodash-es';
  import { useI18n } from 'vue-i18n';
  import { RenderManagerService } from '~/api/modules/rendermanager';
  import MsEditor from '~/components/monaco-editor/ms-editor.vue';
  import useTime from '~/composables/use-time';

  import CreateVersion from '../component-management.vue';

  import type { InputType } from 'bkui-vue/lib/input/input';
  import type { Component } from '~/@types/rendermanager';

  interface IEmit {
    (e: 'refresh'): void;
  }

  interface IProps {
    allowedRange: string;
    row: Component;
  }

  const props = defineProps<IProps>();
  const emits = defineEmits<IEmit>();

  // 引入国际化
  const { t } = useI18n();
  const langMap = {
    label: {
      baseInfo: t('基本信息'),
      ID: t('ID'),
      name: t('名称'),
      currentVersion: t('当前版本'),
      type: t('类型'),
      openScope: t('公开范围'),
      description: t('描述'),
      componentVersion: t('组件版本'),
      version2: t('版本信息'),
      version3: t('版本'),
      importNumber: t('引用数'),
      createTime: t('创建时间'),
      creator: t('创建者'),
      inputParams: t('输入参数'),
      outputParams: t('输出参数'),
      inputParams1: t('输入参数 (Properties)'),
      outputParams1: t('输出参数 (Output)'),
    },
    button: {
      createVersion: t('新建版本'),
      delete: t('删除'),
    },
    tips: {
      deleteVersion: t('最后一个版本无法删除'),
      content2: t('删除后不可恢复'),
      canDelete: t('可删除'),
      notDelete: t('该版本正在被引用，无法删除'),
      notDelete1: t('最后一个版本无法删除'),
      title2: t('确认删除该版本？'),
    },
  };
  const { formatDateString } = useTime();

  const isShow = ref(false);
  const isLoading = ref(false);
  const isUpdateLoading = ref(false);
  const formData = ref({
    displayName: props.row.displayName,
    description: props.row.definition?.description || '',
  });
  const searchValue = ref('');
  const versionList = ref<Component[]>([]);
  const filterVersionList = computed(() => versionList.value.filter(item => item.version.includes(searchValue.value)));
  const currentVersion = ref<Component>({} as Component);
  const inputParamsValue = ref('');
  const outputParamsValue = ref('');

  // ref
  const CreateVersionRef = ref();

  function close() {
    isShow.value = false;
    init();
  }

  // 获取组件信息
  async function getComponentInfo() {
    isLoading.value = true;
    versionList.value = (await RenderManagerService.GetComponent({
      name: props.row.name,
      type: props.row.type,
      withHistory: true,
    }).catch(() => [])) as Component[];
    isLoading.value = false;
  }

  function handleChangeVersion(row: Component) {
    currentVersion.value = row;
  }

  // 组件信息修改后
  async function handleConfirm() {
    const params = {
      ...cloneDeep(props.row),
      displayName: formData.value.displayName,
    };
    params.definition.description = formData.value.description;
    isUpdateLoading.value = true;
    const res = await RenderManagerService.UpdateComponent(params).catch(() => false);
    if (res !== false) {
      formData.value = {
        displayName: params.displayName,
        description: params.definition.description,
      };
      emits('refresh');
    } else {
      init();
    }
    isUpdateLoading.value = false;
  }

  // 新建版本
  function handleCreateVersion() {
    CreateVersionRef.value?.open?.('version');
  }

  // 删除版本
  async function handleDeleteVersion(row: Component) {
    const res = await RenderManagerService.RemoveComponent(row).catch(() => false);
    if (res !== false) {
      getComponentInfo();
      emits('refresh');
    }
  }

  function init() {
    formData.value = {
      displayName: props.row.displayName || '',
      description: props.row.definition?.description || '',
    };
  }

  function open() {
    isShow.value = true;
  }

  watch(
    () => props.row,
    newValue => {
      if (newValue) {
        formData.value = {
          displayName: newValue.displayName || '',
          description: newValue.definition?.description || '',
        };
        getComponentInfo();
      }
    },
  );

  watch(
    versionList,
    newValue => {
      if (newValue.length > 0) {
        [currentVersion.value] = newValue;
      } else {
        currentVersion.value = {} as Component;
      }
    },
    {
      immediate: true,
    },
  );

  watch(currentVersion, newValue => {
    if (!isEmpty(newValue)) {
      inputParamsValue.value = newValue.definition?.properties?.length
        ? yaml.dump(newValue.definition?.properties)
        : '';
      outputParamsValue.value = newValue.output || '';
    }
  });

  defineExpose({
    open,
    close,
  });
</script>

<style lang="postcss" scoped>
  :deep(.bk-form-label) {
    span {
      font-size: 12px;
    }
  }

  :deep(.custom-form-content) {
    .bk-form-content {
      line-height: 12px;
    }
  }

  :deep(.bk-modal-content) {
    scrollbar-gutter: auto !important;
  }

  :deep(.ms-editor) {
    padding-left: 25px;
  }

  :deep(.create-version) {
    padding: 0 3px;
    .bk-button-text {
      width: 100%;
    }
  }
  .version-active,
  .version-content:hover {
    border-color: #3a84ff;
    .version {
      color: #3a84ff;
    }
    .reference-count {
      color: #3a84ff;
      background-color: #e1ecff;
    }
  }
  .version-content:hover {
    .delete-version {
      display: flex;
    }
  }
</style>
