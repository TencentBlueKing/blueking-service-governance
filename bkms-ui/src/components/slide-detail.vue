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
  <div
    ref="modalRef"
    :class="SlideDetailClass['console-slider-detail-modal']"
  >
    <span :class="SlideDetailClass['fold']">
      <span
        :class="[SlideDetailClass['fold-left'], status === 'hidden' && SlideDetailClass['active']]"
        @click="() => toggle(true)"
      >
        <AngleLeft class="text-[22px]" />
      </span>
      <span
        :class="[SlideDetailClass['fold-right'], status === 'expanded' && SlideDetailClass['active']]"
        @click="() => toggle(false)"
      >
        <AngleRight class="text-[22px]" />
      </span>
    </span>
    <div class="absolute left-[-24px] h-full w-[24px] bg-[#f5f7fa]"></div>
    <div
      ref="resizeRef"
      :class="SlideDetailClass['resize-line']"
      @mousedown="handleMouseDown"
    ></div>
    <slot></slot>
  </div>
</template>
<script lang="ts">
  import { defineComponent, ref } from 'vue';

  import { AngleLeft, AngleRight } from 'bkui-vue/lib/icon';

  import SlideDetailClass from './slide-detail.module.css';
  // README: 宽窄表组件
  // 使用方法：要在父盒子设置 relative overflow-x-hidden

  export default defineComponent({
    name: 'SliderDetail',
    components: {
      AngleLeft,
      AngleRight,
    },
    props: {
      modelValue: {
        type: Boolean,
        default: false,
      },
      max: {
        type: Number,
        default: 1200,
      },
      min: {
        type: Number,
        default: 200,
      },
    },
    emits: ['update:modelValue'],
    setup(props, { expose, emit }) {
      const modalRef = ref();
      const resizeRef = ref();

      const status = ref<'expanded' | 'hidden' | 'show'>('show'); // expanded: 全部展开, show: 普通状态 hide: 隐藏状态
      const startX = ref(0);
      const startW = ref(0);
      const containerW = ref(0);

      // 开始拖拽事件
      function handleMouseDown(e: MouseEvent) {
        // 改变分割线颜色
        resizeRef.value.style.borderLeft = '1px solid #3a84ff';
        // 启始位置
        startX.value = e.clientX;
        // 启始宽度
        startW.value = modalRef.value?.clientWidth;
        // 父容器宽度
        containerW.value = modalRef.value?.parentNode?.clientWidth;

        if (!containerW.value) {
          console.warn('parent node is not found');
          return;
        }

        // 注册鼠标拖动事件
        document.addEventListener('mousemove', handleMouseMove);
        // 注册鼠标松开事件
        document.addEventListener('mouseup', handleMouseUp);

        resizeRef.value?.setCapture?.();
      }

      // 鼠标拖动事件
      function handleMouseMove(e: MouseEvent) {
        document.body.style.userSelect = 'none';

        // 结束位置
        const endX = e.clientX;
        // 移动距离
        const moveLen = endX - startX.value;
        // 移动后宽度
        const width = startW.value - moveLen;
        if (width < props.min && moveLen >= 0) {
          hide();
        } else if (width > props.max && moveLen <= 0) {
          expand();
        } else {
          // 设置宽度对应的百分比，最大值为100%
          setWidth(`${Math.min((width / containerW.value) * 100, 100)}%`);
        }
      }
      // 结束拖动事件
      function handleMouseUp() {
        document.body.style.userSelect = '';
        resizeRef.value.style.borderLeft = '';
        resizeRef.value?.releaseCapture?.();
        document.removeEventListener('mousemove', handleMouseMove);
        document.removeEventListener('mouseup', handleMouseUp);
      }

      // 设置面板宽度
      function setWidth(width: string) {
        modalRef.value.style.width = width;
        status.value = 'show';
      }

      // 全屏
      function expand() {
        modalRef.value.style.width = '100%';
        status.value = 'expanded';
        emit('update:modelValue', true);
      }

      // 显示
      function show() {
        setWidth('70%');
        emit('update:modelValue', true);
      }

      // 隐藏
      function hide() {
        modalRef.value.style.width = '0px';
        status.value = 'hidden';
        emit('update:modelValue', false);
      }

      // 切换当前宽窄表状态
      function toggle(isShow: boolean) {
        if (modalRef.value.style.width === '100%' || modalRef.value.style.width === '0px') {
          show();
        } else {
          isShow ? expand() : hide();
        }
      }

      expose({
        show,
        hide,
        expand,
      });

      return {
        show,
        status,
        toggle,
        modalRef,
        resizeRef,
        handleMouseDown,
        SlideDetailClass,
      };
    },
  });
</script>
