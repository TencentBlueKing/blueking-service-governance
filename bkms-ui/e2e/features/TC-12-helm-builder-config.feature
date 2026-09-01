@TC-12 @P0 @config-flow @stateful
Feature: TC-12 Helm 构建配置侧边栏
  作为运维人员
  我需要在 Helm 应用基本信息中查看、编辑并保存构建配置
  以验证 Helm 构建配置侧边栏核心交互与保存链路

  Background:
    Given AccessToken 认证已配置

  @space:default @app:helm @appType:helm
  Scenario: 查看切换并取消 Helm 构建配置编辑
    Given 我在 Helm 应用的基本信息页
    Then Helm 构建配置区域应可见
    And 截图 "01-helm-builder-config-view"
    When 我打开 Helm 构建配置编辑侧栏
    Then Helm 编辑构建配置侧栏应可见
    And 截图 "02-helm-builder-config-slider"
    When 我在 Helm 构建配置侧栏切换来源
    Then Helm 构建配置侧栏应展示当前来源表单
    And 截图 "03-helm-builder-config-switched"
    When 我取消 Helm 构建配置编辑
    Then Helm 编辑构建配置侧栏应关闭
    And 截图 "04-helm-builder-config-cancelled"

  @space:default @app:helm @appType:helm
  Scenario: 校验并保存有效 Helm 构建配置
    Given 我在 Helm 应用的基本信息页
    When 我打开 Helm 构建配置编辑侧栏
    Then Helm 编辑构建配置侧栏应可见
    When 我提交无效 Helm 构建配置
    Then Helm 构建配置侧栏应展示必填校验提示
    And 截图 "05-helm-builder-config-validation"
    When 我保存有效 Helm 构建配置
    Then Helm 构建配置保存成功并关闭侧栏
    And 截图 "06-helm-builder-config-saved"
