/**
 * 测试专用的「接口失败」哨兵错误。
 *
 * 业务约定：HTTP 错误由 fetch interceptor 统一拦截反馈，业务层不手写 catch。
 * 但测试里 API 被 vi.mock，没有 interceptor 层，rejection 会从事件处理器逃逸成
 * unhandled rejection。用例用本类抛错，setup.ts 的分类器即可按类型判定为「预期」，
 * 无需按错误消息文案匹配（文案是业务自定义的，无法穷举）。
 */
export class MockedApiError extends Error {
  constructor(message = 'mocked api error') {
    super(message);
    this.name = 'MockedApiError';
  }
}
