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
    draggable
    :footer-align="'center'"
    :header-align="'center'"
    render-directive="if"
    theme="primary"
    :title="title"
    :width="width"
  >
    <slot></slot>
    <template #footer>
      <Button
        class="mr-[10px]"
        :loading="loading"
        theme="danger"
        @click="submit"
      >
        {{ t('确定') }}
      </Button>
      <Button
        :disabled="loading"
        @click="isShow = false"
        >{{ t('取消') }}</Button
      >
    </template>
  </Dialog>
</template>

<script lang="ts" setup>
  import { Button, Dialog } from 'bkui-vue';
  import { useI18n } from 'vue-i18n';

  interface Emits {
    (e: 'confirm'): void;
  }
  const isShow = defineModel('isShow', { type: Boolean });
  defineProps({
    title: {
      type: String,
      default: '',
    },
    width: {
      type: Number,
      default: 520,
    },
    loading: {
      type: Boolean,
      default: false,
    },
  });
  const emit = defineEmits<Emits>();

  // 国际化
  const { t } = useI18n();

  function submit() {
    emit('confirm');
  }
</script>
