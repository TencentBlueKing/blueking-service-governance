@TC-07 @P0 @config-flow @stateful
Feature: TC-07 更新策略配置
  作为运维人员
  我需要在应用配置中查看、编辑并保存部署更新策略
  以验证更新策略核心交互与格式校验能力

  Background:
    Given AccessToken 认证已配置

  @space:default @app:default
  Scenario: 查看编辑校验并保存测试环境更新策略
    Given 我在当前应用的测试环境部署配置页
    Then 更新策略区域应展示当前配置
    And 截图 "01-update-strategy-view"
    When 我编辑更新策略后取消
    And 截图 "02-update-strategy-cancelled"
    Then 更新策略不应包含取消测试值
    When 我提交无效更新策略配置
    And 截图 "03-update-strategy-invalid"
    Then 更新策略应展示格式校验提示
    When 我保存有效更新策略配置
    And 截图 "04-update-strategy-saved"
    Then 更新策略应处于查看态
    And 更新策略应展示已保存配置
