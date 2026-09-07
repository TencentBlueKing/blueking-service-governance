@TC-09 @P0 @config-flow @stateful
Feature: TC-09 开发模式
  作为开发人员
  我需要在应用配置中开启或关闭测试环境开发模式
  以验证开发模式确认弹窗、状态切换与操作步骤展示能力

  Background:
    Given AccessToken 认证已配置

  @space:default @app:default
  Scenario: 查看取消开启并开关测试环境开发模式
    Given 我在当前应用的测试环境部署配置页
    When 我确保开发模式已关闭
    Then 开发模式区域应展示关闭状态
    And 截图 "01-dev-mode-off"
    When 我尝试开启开发模式后取消
    And 截图 "02-dev-mode-enable-cancelled"
    Then 开发模式应保持关闭
    When 我确认开启开发模式
    And 截图 "03-dev-mode-enabled"
    Then 开发模式应展示开启后的操作步骤
    When 我刷新页面并切换到测试环境
    And 截图 "04-dev-mode-enabled-after-refresh"
    Then 开发模式应展示开启后的操作步骤
    When 我确认关闭开发模式
    And 截图 "05-dev-mode-disabled"
    Then 开发模式应保持关闭
