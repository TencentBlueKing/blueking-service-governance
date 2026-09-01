@TC-10 @P0 @config-flow @stateful
Feature: TC-10 生命周期配置
  作为运维人员
  我需要在应用配置中查看、编辑并保存生命周期配置
  以验证 preStop 与优雅退出时间的核心交互和校验能力

  Background:
    Given AccessToken 认证已配置

  @space:default @app:default
  Scenario: 查看编辑校验并保存默认生命周期
    Given 我在当前应用的默认部署配置页
    Then 生命周期区域应展示当前配置
    And 截图 "01-lifecycle-view"
    When 我编辑生命周期后取消
    And 截图 "02-lifecycle-cancelled"
    Then 生命周期不应包含取消测试命令
    When 我选择生命周期自定义命令选项
    And 截图 "03-lifecycle-custom-command"
    Then 生命周期应展示自定义命令输入项
    When 我提交无效生命周期配置
    And 截图 "04-lifecycle-invalid"
    Then 生命周期应展示必填校验提示
    When 我保存有效生命周期配置
    And 截图 "05-lifecycle-saved"
    Then 生命周期应处于查看态
    And 生命周期应展示已保存配置
