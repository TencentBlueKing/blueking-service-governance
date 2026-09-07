@TC-11 @P0 @config-flow @stateful
Feature: TC-11 tRPC 构建配置侧边栏
  作为运维人员
  我需要在基本信息中查看、编辑并保存构建配置
  以验证构建配置侧边栏核心交互与保存链路

  Background:
    Given AccessToken 认证已配置

  @space:default @app:trpc @appType:trpc
  Scenario: 查看切换并取消构建配置编辑
    Given 我在 tRPC 应用的基本信息页
    Then 构建配置区域应可见
    And 截图 "01-builder-config-view"
    When 我打开构建配置编辑侧栏
    Then 编辑构建配置侧栏应可见
    And 截图 "02-builder-config-slider"
    When 我在构建配置侧栏切换来源
    Then 构建配置侧栏应展示当前来源表单
    And 截图 "03-builder-config-switched"
    When 我取消构建配置编辑
    Then 编辑构建配置侧栏应关闭
    And 截图 "04-builder-config-cancelled"

  @space:default @app:trpc @appType:trpc
  Scenario: 校验并保存有效构建配置
    Given 我在 tRPC 应用的基本信息页
    When 我打开构建配置编辑侧栏
    Then 编辑构建配置侧栏应可见
    When 我提交无效构建配置
    Then 构建配置侧栏应展示必填校验提示
    And 截图 "05-builder-config-validation"
    When 我保存有效构建配置
    Then 构建配置保存成功并关闭侧栏
    And 截图 "06-builder-config-saved"
