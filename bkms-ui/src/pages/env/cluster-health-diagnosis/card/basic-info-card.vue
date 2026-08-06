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
  <div class="px-[32px] py-[24px] min-w-[520px] h-[252px] bg-[#fff]">
    <span class="text-[#313238] text-[14px] font-bold">{{ $t('基础信息') }}</span>
    <div class="grid grid-cols-2 gap-y-[24px] pt-[24px]">
      <FieldItem
        container-height="auto"
        :field-value="$t('集群 ID')"
        :field-width="70"
        :value="clusterInfo?.clusterID"
        value-color="#313238"
      >
      </FieldItem>
      <FieldItem
        container-height="auto"
        :field-value="$t('集群名称')"
        :field-width="70"
        :value="clusterInfo?.clusterName"
        value-color="#313238"
      >
      </FieldItem>
      <FieldItem
        container-height="auto"
        :field-value="$t('K8s 版本')"
        :field-width="70"
        :value="clusterInfo?.clusterVersion"
        value-color="#313238"
      >
      </FieldItem>
      <FieldItem
        container-height="auto"
        :field-value="$t('区域')"
        :field-width="70"
        :value="clusterInfo?.region"
        value-color="#313238"
      >
      </FieldItem>
      <FieldItem
        class="col-span-2"
        container-height="auto"
        :field-value="$t('集群类型')"
        :field-width="70"
        :value="clusterInfo?.clusterType"
        value-color="#313238"
      >
      </FieldItem>
      <FieldItem
        container-height="auto"
        field-value="VPC ID"
        :field-width="70"
        :value="clusterInfo?.vpcID"
        value-color="#313238"
      >
      </FieldItem>
      <FieldItem
        container-height="auto"
        :field-value="$t('集群 CIDR')"
        :field-width="70"
        :value="clusterInfo?.networkParams?.cidrs?.join(', ')"
        value-color="#313238"
      >
      </FieldItem>
    </div>
  </div>
</template>

<script lang="ts" setup>
  import { ClusterReportOutput } from '~/@types/v1/bkintegrations-kubeinsight';

  interface IProps {
    clusterInfo?: ClusterReportOutput['clusterInfo'];
  }
  defineProps<IProps>();
</script>
