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
  <div>
    <Form.FormItem
      label="最小CPU"
      property="cpu"
      required
    >
      <Input
        :min="0"
        placeholder="请输入"
        suffix="核"
        type="number"
        :value="modelValue"
        @input="handleChange"
      />
    </Form.FormItem>
    <Form.FormItem
      label="最小内存"
      property="memory"
      required
    >
      <Input
        :min="0"
        placeholder="请输入"
        suffix="GB"
        type="number"
        :value="modelValue"
        @input="handleChange"
      />
    </Form.FormItem>
    <Form.FormItem
      label="存储类型"
      property="memoryType"
      required
    >
      <Select
        :filterable="false"
        placeholder="请选择存储类型"
        :value="modelValue"
        @change="handleChange"
      >
        <Select.Option
          id="ssd"
          name="SSD"
        ></Select.Option>
        <Select.Option
          id="hhd"
          name="HHD"
        ></Select.Option>
      </Select>
    </Form.FormItem>
    <Form.FormItem
      label="最小实例"
      property="minInstance"
      required
    >
      <Input
        :min="0"
        placeholder="请输入"
        type="number"
        :value="modelValue"
        @input="handleChange"
      />
    </Form.FormItem>
    <Form.FormItem
      label="最大实例"
      property="maxInstance"
      required
    >
      <Input
        :min="0"
        placeholder="请输入"
        type="number"
        :value="modelValue"
        @input="handleChange"
      />
    </Form.FormItem>
    <Form.FormItem
      label="扩容条件"
      property="scaleCondition"
      required
    >
      <Input
        placeholder="请输入"
        :value="modelValue"
        @input="handleChange"
      />
    </Form.FormItem>
    <Form.FormItem
      label="缩容条件"
      property="shrinkCondition"
      required
    >
      <Input
        placeholder="请输入"
        :value="modelValue"
        @input="handleChange"
      />
    </Form.FormItem>
    <Form.FormItem
      label="最大宽带"
      property="broadband"
      required
    >
      <Input
        :min="0"
        placeholder="请输入"
        suffix="Mbps"
        type="number"
        :value="modelValue"
        @input="handleChange"
      />
    </Form.FormItem>
    <Form.FormItem
      label="访问控制"
      property="access"
      required
    >
      <Select
        :filterable="false"
        placeholder="请选择"
        :value="modelValue"
        @change="handleChange"
      >
        <Select.Option
          id="0"
          name="公开"
        ></Select.Option>
        <Select.Option
          id="1"
          name="仅内网"
        ></Select.Option>
        <Select.Option
          id="2"
          name="特定IP"
        ></Select.Option>
      </Select>
    </Form.FormItem>
    <Form.FormItem
      label="数据加密"
      property="encrypt"
      required
    >
      <Select
        :filterable="false"
        placeholder="请选择"
        :value="modelValue"
        @change="handleChange"
      >
        <Select.Option
          id="0"
          name="启用"
        ></Select.Option>
        <Select.Option
          id="1"
          name="禁用"
        ></Select.Option>
      </Select>
    </Form.FormItem>
    <Form.FormItem
      label="自动备份"
      property="backups"
      required
    >
      <Select
        :filterable="false"
        placeholder="请选择"
        :value="modelValue"
        @change="handleChange"
      >
        <Select.Option
          id="0"
          name="每日"
        ></Select.Option>
        <Select.Option
          id="1"
          name="每周"
        ></Select.Option>
        <Select.Option
          id="2"
          name="禁用"
        ></Select.Option>
      </Select>
    </Form.FormItem>
    <Form.FormItem
      label="保留天数"
      property="retentionDays"
      required
    >
      <Input
        :min="0"
        placeholder="请输入"
        suffix="天"
        type="number"
        :value="modelValue"
        @input="handleChange"
      />
    </Form.FormItem>
  </div>
</template>
<script setup lang="ts">
  import { Form, Input, Select } from 'bkui-vue';

  interface IProps {
    modelValue: string;
    property: string;
  }

  defineProps<IProps>();
  const emit = defineEmits(['update:modelValue']);

  function handleChange(value: string) {
    emit('update:modelValue', value);
  }
</script>
