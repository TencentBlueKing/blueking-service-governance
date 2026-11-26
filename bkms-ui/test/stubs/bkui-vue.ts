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
import { defineComponent, h, inject, onBeforeUnmount, onMounted, provide, reactive, watch } from 'vue';
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
/**
 * Form/FormItem 的最小校验契约：
 * FormItem 向所属 Form 注册 property/rules，Form.validate() 同步执行
 * validator/required 规则并把失败消息渲染到 FormItem，对齐真实组件的
 * 校验行为（trigger 差异不模拟，调用 validate 时全量执行）。
 */
interface FormFieldRule {
  message?: string;
  required?: boolean;
  trigger?: string;
  validator?: (value: unknown) => boolean | string;
}

interface FormStubContext {
  fields: FormStubField[];
  getModel: () => Record<string, unknown> | undefined;
}

interface FormStubField {
  error: { message: string };
  property: string;
  rules: FormFieldRule[];
}

const formStubKey: InjectionKey<FormStubContext> = Symbol('FormStubContext');

function getValueByPath(source: Record<string, unknown> | undefined, path: string) {
  return path
    .split('.')
    .reduce<unknown>((acc, key) => (acc as Record<string, unknown> | undefined)?.[key as string], source);
}

const FormItem = defineComponent({
  name: 'FormItemStub',
  props: {
    label: { type: String, default: '' },
    property: { type: String, default: '' },
    required: { type: Boolean, default: false },
    rules: { type: Array as PropType<FormFieldRule[]>, default: () => [] },
  },
  setup(props, { slots }) {
    const context = inject(formStubKey, null);
    const field = reactive<FormStubField>({ property: props.property, rules: props.rules, error: { message: '' } });
    watch(
      () => [props.property, props.rules] as const,
      ([property, rules]) => {
        field.property = property;
        field.rules = rules;
      },
    );
    onMounted(() => context?.fields.push(field));
    onBeforeUnmount(() => {
      const index = context?.fields.indexOf(field) ?? -1;
      if (index >= 0) context?.fields.splice(index, 1);
    });
    return () =>
      h('div', { class: 'bk-form-item' }, [
        props.label
          ? h('label', { class: 'bk-form-label' }, [
              props.required ? h('i', { class: 'required' }, '*') : null,
              props.label,
            ])
          : null,
        h('div', { class: 'bk-form-content' }, slots.default?.()),
        field.error.message ? h('p', { class: 'bk-form-error' }, field.error.message) : null,
      ]);
  },
});

const FormComponent = defineComponent({
  name: 'FormStub',
  props: { model: { type: Object as PropType<Record<string, unknown>>, default: undefined } },
  setup(props, { expose, slots }) {
    const fields: FormStubField[] = [];
    provide(formStubKey, { fields, getModel: () => props.model });
    expose({
      clearValidate: () => fields.forEach(field => (field.error.message = '')),
      validate: async () => {
        let valid = true;
        for (const field of fields) {
          const value = getValueByPath(props.model, field.property);
          for (const rule of field.rules) {
            let failed = false;
            if (typeof rule.validator === 'function') {
              failed = rule.validator(value) !== true;
            } else if (rule.required) {
              failed = value === undefined || value === null || `${value}`.trim() === '';
            }
            field.error.message = failed ? rule.message || '' : '';
            if (failed) {
              valid = false;
              break;
            }
          }
        }
        return valid;
      },
    });
    return () => h('form', slots.default?.());
  },
});

export const Button = Object.assign(
  defineComponent({
    name: 'ButtonStub',
    props: { disabled: { type: Boolean, default: false } },
    setup(props, { slots, attrs }) {
      return () => h('button', { ...attrs, type: 'button', disabled: props.disabled || undefined }, slots.default?.());
    },
  }),
  { ButtonGroup: createEmptyStub() },
);
const OverflowTitle = defineComponent({
  name: 'OverflowTitleStub',
  setup(_, { slots }) {
    return () => h('span', slots.default?.());
  },
});

export const Checkbox = createEmptyStub();
export { Dialog };
export const Form = Object.assign(FormComponent, { FormItem });
export { FormItem };
const Input = defineComponent({
  name: 'InputStub',
  props: {
    modelValue: { type: [String, Number] as PropType<number | string>, default: '' },
    type: { type: String, default: 'text' },
  },
  emits: ['update:modelValue', 'enter'],
  setup(props, { emit, attrs }) {
    return () =>
      h(props.type === 'textarea' ? 'textarea' : 'input', {
        ...attrs,
        // 对齐真实 Input 的默认占位文案契约。
        placeholder: (attrs.placeholder as string) ?? '请输入',
        ...(props.type === 'textarea' ? {} : { type: props.type }),
        value: props.modelValue,
        onInput: (e: Event) => emit('update:modelValue', (e.target as HTMLInputElement).value),
      });
  },
});
export const Radio = Object.assign(RadioGroup, { Group: RadioGroup, Button: RadioButton });
const ResizeLayout = defineComponent({
  name: 'ResizeLayoutStub',
  setup(_, { slots, expose }) {
    // 真实组件以 asideRef 控制侧栏显隐，composable 的 hideOrShowError 依赖该结构。
    expose({ asideRef: { hidden: true } });
    return () => h('div', { 'data-testid': 'resize-layout' }, [slots.default?.(), slots.main?.(), slots.aside?.()]);
  },
});
export const Select = Object.assign(createEmptyStub(), { Group: createEmptyStub(), Option: createEmptyStub() });
export { Input, ResizeLayout };
export { Popover, Sideslider };
export { OverflowTitle };
export const Tag = createEmptyStub();
export const Dropdown = Object.assign(createEmptyStub(), {
  DropdownMenu: createEmptyStub(),
  DropdownItem: createEmptyStub(),
});
