/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
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

const HTTP_METHODS = new Set(['get', 'post', 'put', 'delete', 'patch']);
const METHOD_PREFIX_MAP = {
  delete: 'delete',
  get: 'get',
  patch: 'patch',
  post: 'create',
  put: 'update',
};

function collectApiImportTypes(operations) {
  const types = new Set();
  for (const operation of operations) {
    types.add(operation.requestTypeName);
    collectImportTypeNames(operation.defaultResponseTypeName, types);
  }
  return [...types];
}

function collectImportTypeNames(typeText, types) {
  if (!typeText) return;

  const withoutStringLiterals = String(typeText).replace(/"[^"]*"/g, '');
  for (const match of withoutStringLiterals.matchAll(/\b[A-Z][A-Za-z0-9_]*\b/g)) {
    const typeName = match[0];
    if (typeName !== 'Record') {
      types.add(typeName);
    }
  }
}

function collectOperationsByTag(swagger) {
  const operationsByTag = new Map();

  for (const [urlPath, pathItem] of Object.entries(swagger.paths || {})) {
    for (const [method, operation] of Object.entries(pathItem || {})) {
      if (!HTTP_METHODS.has(method)) continue;

      const tag = normalizeTag(operation.tags?.[0] || 'default');
      if (!operationsByTag.has(tag)) operationsByTag.set(tag, []);
      operationsByTag.get(tag).push({
        method,
        path: urlPath,
        operation,
        requestTypeName: `${upperFirst(resolveOperationName(method, urlPath, operation))}Request`,
        responseTypeName: resolveResponseTypeName(operation),
        defaultResponseTypeName: resolveDefaultResponseTypeName(swagger, operation),
        methodName: resolveOperationName(method, urlPath, operation),
      });
    }
  }

  return operationsByTag;
}

function collectRefsFromSchema(schema, refs) {
  if (!schema || typeof schema !== 'object') return;

  if (schema.$ref) {
    refs.add(refToName(schema.$ref));
  }

  if (schema.items) collectRefsFromSchema(schema.items, refs);
  if (schema.additionalProperties && schema.additionalProperties !== true) {
    collectRefsFromSchema(schema.additionalProperties, refs);
  }
  for (const item of schema.allOf || []) collectRefsFromSchema(item, refs);
  for (const propertySchema of Object.values(schema.properties || {})) collectRefsFromSchema(propertySchema, refs);
}

function collectRelatedDefinitions(swagger, operations) {
  const related = new Set();

  for (const item of operations) {
    const parameters = item.operation.parameters || [];
    for (const parameter of parameters) {
      collectRefsFromSchema(parameter.schema, related);
    }

    const successSchema = getSuccessResponseSchema(item.operation);
    collectRefsFromSchema(successSchema, related);
  }

  const pending = [...related];
  while (pending.length > 0) {
    const definitionName = pending.pop();
    const schema = swagger.definitions?.[definitionName];
    if (!schema) continue;

    const nestedRefs = new Set();
    collectRefsFromSchema(schema, nestedRefs);
    for (const refName of nestedRefs) {
      if (!related.has(refName)) {
        related.add(refName);
        pending.push(refName);
      }
    }
  }

  return [...related].filter(name => Boolean(swagger.definitions?.[name]));
}

function docTextLines(text, starIndent = 1) {
  if (!text) return [];
  const spaces = ' '.repeat(starIndent - 1);
  return String(text)
    .split('\n')
    .map(line => `${spaces} * ${line.trim()}`.trimEnd());
}

function generateApiFile(swagger, tag, operations) {
  const serviceName = `${upperFirst(toCamelCase(tag))}Service`;
  const lines = [
    '/* eslint-disable */',
    '// gen-api-v1.js 自动生成，请勿手动修改',
    '// 来源：apps/bkms-server/docs/apis/swagger.json',
    `// Swagger：${swagger.info?.title || ''} ${swagger.info?.version || ''}`.trim(),
    `// BasePath：${swagger.basePath || ''}`,
    "import type { Config } from '~/api/interceptors';",
    "import type { NoInfer } from '~/api/ts-helpers';",
    "import { v1Fetch } from '~/api/clients';",
    `import type { ${collectApiImportTypes(operations).join(', ')} } from '~/@types/v1/${tag}';`,
    '',
    `export const ${serviceName} = {`,
  ];

  for (const operation of operations) {
    lines.push(...generateOperationLines(operation));
  }

  lines.push('};', '');
  return `${lines
    .join('\n')
    .replace(/\n{3,}/g, '\n\n')
    .trim()}\n`;
}

function generateApiIndexFile(tags) {
  return `${tags.map(tag => `export * from './${tag}';`).join('\n')}\n`;
}

function generateDocLines(text, indent = 0) {
  const content = docTextLines(text, indent + 1);
  if (content.length === 0) return [];

  const spaces = ' '.repeat(indent);
  return [`${spaces}/**`, ...content, `${spaces} */`];
}

function generateFromSwagger(swagger) {
  const operationsByTag = collectOperationsByTag(swagger);
  const typeFiles = {};
  const apiFiles = {};

  for (const [tag, operations] of operationsByTag) {
    const relatedDefinitions = collectRelatedDefinitions(swagger, operations);
    const typeFileName = `${tag}.d.ts`;
    typeFiles[typeFileName] = generateTypeFile(swagger, tag, operations, relatedDefinitions);
    apiFiles[`${tag}.ts`] = generateApiFile(swagger, tag, operations);
  }

  apiFiles['index.ts'] = generateApiIndexFile([...operationsByTag.keys()]);

  return {
    apiFiles,
    typeFiles,
  };
}

function generateOperationLines(operationItem) {
  const { method, path, operation, requestTypeName, defaultResponseTypeName, methodName } = operationItem;
  const responseType = defaultResponseTypeName || 'unknown';
  const lines = ['  /**', ...docTextLines(operation.summary, 3)];

  if (operation.description) {
    lines.push('   *');
    lines.push(...docTextLines(operation.description, 3));
  }

  lines.push('   *');
  lines.push(`   * @method ${method.toUpperCase()}`);
  lines.push(`   * @path ${path}`);
  lines.push(`   * @tag ${(operation.tags || []).join(', ')}`);

  for (const parameter of operation.parameters || []) {
    const type = parameter.in === 'body' ? schemaToTsType(parameter.schema) : schemaToTsType(parameter);
    const required = parameter.required ? ' required' : '';
    const description = parameter.description ? ` ${parameter.description}` : '';
    lines.push(`   * @param ${parameter.name} ${parameter.in} ${type}${required}${description}`);
  }

  for (const [statusCode, response] of Object.entries(operation.responses || {})) {
    const type = schemaToTsType(response.schema);
    lines.push(`   * @response ${statusCode} ${type}${response.description ? ` ${response.description}` : ''}`);
  }

  lines.push('   */');
  // 使用 NoInfer：避免从实参推断 Request，否则会放宽为“多写字段也合法”，丢失多余属性检查
  lines.push(
    `  ${methodName}: async <Request extends ${requestTypeName} = ${requestTypeName}, ResponseData = ${responseType}>(`,
  );
  lines.push('    params?: NoInfer<Request>,');
  lines.push('    config?: Config,');
  lines.push(`  ) => await v1Fetch.${method}<Request, ResponseData>('${path}')(params, config),`);

  return lines;
}

function generatePropertyLines(name, schema, required) {
  const lines = [];
  lines.push(...generateDocLines(schema.description, 2));
  lines.push(`  ${quotePropertyName(name)}${required ? '' : '?'}: ${schemaToTsType(schema)};`);
  return lines;
}

function generateRequestType(operationItem) {
  const pathAndQueryParams = (operationItem.operation.parameters || []).filter(parameter => parameter.in !== 'body');
  const bodyParam = (operationItem.operation.parameters || []).find(parameter => parameter.in === 'body');
  const bodyType = schemaToTsType(bodyParam?.schema);
  const requestTypeName = operationItem.requestTypeName;

  if (bodyParam && bodyType !== 'unknown' && pathAndQueryParams.length > 0) {
    return [
      `export type ${requestTypeName} = ${bodyType} & {`,
      ...pathAndQueryParams.flatMap(parameter =>
        generatePropertyLines(parameter.name, parameter, Boolean(parameter.required)),
      ),
      '};',
    ];
  }

  if (bodyParam && bodyType !== 'unknown') {
    return [`export type ${requestTypeName} = ${bodyType};`];
  }

  return [
    `export interface ${requestTypeName} {`,
    ...pathAndQueryParams.flatMap(parameter =>
      generatePropertyLines(parameter.name, parameter, Boolean(parameter.required)),
    ),
    '}',
  ];
}

function generateSchemaType(definitionName, schema) {
  const typeName = toTypeName(definitionName);

  if (schema.enum?.length) {
    return [
      ...generateDocLines(schema.description),
      `export type ${typeName} = ${schema.enum.map(value => JSON.stringify(value)).join(' | ')};`,
    ];
  }

  if (schema.type !== 'object' && !schema.properties && !schema.allOf) {
    return [...generateDocLines(schema.description), `export type ${typeName} = ${schemaToTsType(schema)};`];
  }

  const normalizedSchema = mergeAllOf(schema);
  const required = new Set(normalizedSchema.required || []);
  const properties = normalizedSchema.properties || {};

  return [
    ...generateDocLines(schema.description),
    `export interface ${typeName} {`,
    ...Object.entries(properties).flatMap(([propertyName, propertySchema]) =>
      generatePropertyLines(propertyName, propertySchema, required.has(propertyName)),
    ),
    '}',
  ];
}

function generateTypeFile(swagger, tag, operations, relatedDefinitions) {
  const lines = [
    '/* eslint-disable */',
    '// gen-api-v1.js 自动生成，请勿手动修改',
    '// 来源：apps/bkms-server/docs/apis/swagger.json',
    `// 模块：${tag}`,
    '',
  ];

  for (const operation of operations) {
    lines.push(...generateRequestType(operation));
    lines.push('');
  }

  for (const definitionName of relatedDefinitions) {
    const schema = swagger.definitions[definitionName];
    lines.push(...generateSchemaType(definitionName, schema));
    lines.push('');
  }

  return `${lines
    .join('\n')
    .replace(/\n{3,}/g, '\n\n')
    .trim()}\n`;
}

function getSuccessResponseSchema(operation) {
  return (
    operation.responses?.[200]?.schema || operation.responses?.['200']?.schema || operation.responses?.default?.schema
  );
}

function lowerFirst(value) {
  return (
    String(value || '')
      .charAt(0)
      .toLowerCase() + String(value || '').slice(1)
  );
}

function mergeAllOf(schema) {
  if (!schema.allOf?.length) return schema;

  return schema.allOf.reduce(
    (merged, item) => {
      const normalized = mergeAllOf(item);
      return {
        ...merged,
        ...normalized,
        properties: {
          ...(merged.properties || {}),
          ...(normalized.properties || {}),
        },
        required: [...(merged.required || []), ...(normalized.required || [])],
      };
    },
    {
      ...schema,
      allOf: undefined,
    },
  );
}

function normalizeTag(tag) {
  return String(tag || 'default').replace(/[^\w-]/g, '_');
}

function quotePropertyName(name) {
  return /^[a-zA-Z_$][\w$]*$/.test(name) ? name : JSON.stringify(name);
}

function refToName(ref) {
  return String(ref || '').replace('#/definitions/', '');
}

function resolveDefaultResponseTypeName(swagger, operation) {
  const successSchema = getSuccessResponseSchema(operation);
  const normalizedSchema = resolveSchema(swagger, successSchema);
  const dataSchema = normalizedSchema?.properties?.data;

  if (dataSchema) {
    return schemaToTsType(dataSchema);
  }

  return schemaToTsType(successSchema);
}

/**
 * 导出方法名 / 请求类型前缀：优先使用 Swagger `operationId`（转 camelCase），
 * 缺失或空白时退回由 HTTP 方法 + path 推导（与历史行为一致）。
 */
function resolveOperationName(method, path, operation) {
  const operationId = operation?.operationId;
  if (typeof operationId === 'string' && operationId.trim() !== '') {
    return toCamelCase(operationId.trim());
  }

  return resolveOperationNameFromPath(method, path);
}

function resolveOperationNameFromPath(method, path) {
  const prefix = METHOD_PREFIX_MAP[method] || method;
  const parts = path
    .split('/')
    .filter(Boolean)
    .filter(segment => !segment.startsWith('{'))
    .map((segment, index, list) => {
      if (index === 0 && path.split('/').filter(Boolean)[1]?.startsWith('{')) {
        return singularize(segment);
      }
      if (index === list.length - 1 && segment === 'public-vars') return 'public-vars';
      return segment;
    });

  return toCamelCase([prefix, ...parts].join('-'));
}

function resolveResponseTypeName(operation) {
  return schemaToTsType(getSuccessResponseSchema(operation));
}

function resolveSchema(swagger, schema) {
  if (!schema || typeof schema !== 'object') return schema;
  if (schema.$ref) return swagger.definitions?.[refToName(schema.$ref)] || schema;
  return mergeAllOf(schema);
}

function schemaToTsType(schema) {
  if (!schema) return 'unknown';
  if (schema.$ref) return toTypeName(schema.$ref);
  if (schema.allOf?.length) return schema.allOf.map(item => schemaToTsType(item)).join(' & ');
  if (schema.enum?.length) return schema.enum.map(value => JSON.stringify(value)).join(' | ');

  switch (schema.type) {
    case 'array':
      return `${schemaToTsType(schema.items)}[]`;
    case 'boolean':
      return 'boolean';
    case 'integer':
    case 'number':
      return 'number';
    case 'object': {
      if (schema.additionalProperties) {
        const valueType =
          schema.additionalProperties === true ? 'unknown' : schemaToTsType(schema.additionalProperties);
        return `Record<string, ${valueType}>`;
      }
      if (schema.properties) {
        return `{\n${Object.entries(schema.properties)
          .flatMap(([name, propertySchema]) => generatePropertyLines(name, propertySchema, false))
          .join('\n')}\n}`;
      }
      return 'Record<string, unknown>';
    }
    case 'string':
      return 'string';
    default:
      return 'unknown';
  }
}

function singularize(value) {
  return String(value).endsWith('s') ? String(value).slice(0, -1) : String(value);
}

function toCamelCase(value) {
  const words = String(value)
    .replace(/[{}]/g, '')
    .split(/[^a-zA-Z0-9]+/)
    .filter(Boolean);

  return words
    .map((word, index) => {
      const normalized = word.replace(/ID$/, 'Id');
      return index === 0 ? lowerFirst(normalized) : upperFirst(normalized);
    })
    .join('');
}

function toTypeName(value) {
  const name = refToName(value);
  return upperFirst(toCamelCase(name.split('.').pop()));
}

function upperFirst(value) {
  return (
    String(value || '')
      .charAt(0)
      .toUpperCase() + String(value || '').slice(1)
  );
}

module.exports = {
  generateFromSwagger,
};
