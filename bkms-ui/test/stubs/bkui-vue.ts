/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License; you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */
import { defineComponent, h, inject, provide } from 'vue';
import type { InjectionKey, PropType } from 'vue';

const createEmptyStub = () => defineComponent({ name: 'BkuiStub', template: '<span><slot /></span>' });
const Popover = defineComponent({
  name: 'PopoverStub',
  setup(_, { slots }) {
    return () => h('div', { 'data-testid': 'popover' }, [slots.default?.(), slots.content?.()]);
  },
});
const Dialog = defineComponent({
  name: 'DialogStub',
  props: { isShow: Boolean },
  setup(props, { slots }) {
    return () => (props.isShow ? h('div', { 'data-testid': 'dialog' }, [slots.default?.(), slots.footer?.()]) : null);
  },
});
const Sideslider = defineComponent({
  name: 'SidesliderStub',
  props: { isShow: { type: Boolean, default: false } },
  setup(props, { slots }) {
    return () =>
      props.isShow ? h('div', { 'data-testid': 'sideslider' }, [slots.default?.(), slots.footer?.()]) : null;
  },
});

interface RadioGroupContext {
  change: (value: boolean | number | string) => void;
  modelValue: () => boolean | number | string;
}
const radioGroupKey: InjectionKey<RadioGroupContext> = Symbol('RadioGroupStub');
const RadioGroup = defineComponent({
  name: 'RadioGroupStub',
  props: { modelValue: { type: [String, Number, Boolean] as PropType<boolean | number | string>, default: '' } },
  emits: ['update:modelValue'],
  setup(props, { emit, slots }) {
    provide(radioGroupKey, {
      modelValue: () => props.modelValue,
      change: (value: boolean | number | string) => emit('update:modelValue', value),
    });
    return () => h('div', { role: 'radiogroup' }, slots.default?.());
  },
});
const RadioButton = defineComponent({
  name: 'RadioButtonStub',
  props: { label: { type: [String, Number, Boolean] as PropType<boolean | number | string>, required: true } },
  setup(props, { slots }) {
    const group = inject(radioGroupKey, null);
    return () =>
      h('label', { class: 'bk-radio-button' }, [
        h('input', {
          checked: group ? group.modelValue() === props.label : false,
          type: 'radio',
          onChange: () => group?.change(props.label),
        }),
        h('span', slots.default?.()),
      ]);
  },
});
const FormItem = defineComponent({
  name: 'FormItemStub',
  props: { label: { type: String, default: '' }, required: { type: Boolean, default: false } },
  setup(props, { slots }) {
    return () =>
      h('div', { class: 'bk-form-item' }, [
        props.label
          ? h('label', { class: 'bk-form-label' }, [
              props.required ? h('i', { class: 'required' }, '*') : null,
              props.label,
            ])
          : null,
        h('div', { class: 'bk-form-content' }, slots.default?.()),
      ]);
  },
});

export const Button = createEmptyStub();
export const Checkbox = createEmptyStub();
export { Dialog };
export const Form = Object.assign(createEmptyStub(), { FormItem });
export { FormItem };
export const Input = createEmptyStub();
export const Radio = Object.assign(RadioGroup, { Group: RadioGroup, Button: RadioButton });
export const Select = Object.assign(createEmptyStub(), { Group: createEmptyStub(), Option: createEmptyStub() });
export { Popover, Sideslider };
export const Tag = createEmptyStub();
export const Dropdown = Object.assign(createEmptyStub(), {
  DropdownMenu: createEmptyStub(),
  DropdownItem: createEmptyStub(),
});
