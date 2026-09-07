@TC-08 @P0 @config-flow @stateful
Feature: TC-08 元数据配置
  作为运维人员
  我需要在应用配置中查看、编辑并保存部署元数据
  以验证 Labels 和 Annotations 的配置、校验与回显能力

  Background:
    Given AccessToken 认证已配置

  @space:default @app:default
  Scenario: 查看编辑校验并保存测试环境元数据配置
    Given 我在当前应用的测试环境部署配置页
    Then 元数据配置区域应展示标签和注解卡片
    And 截图 "01-metadata-view"
    When 我编辑标签元数据后取消
    And 截图 "02-metadata-label-cancelled"
    Then 标签元数据不应包含取消测试值
    When 我提交系统保留标签配置
    And 截图 "03-metadata-label-reserved"
    Then 元数据配置应展示系统保留字段校验提示
    When 我保存有效标签元数据配置
    And 截图 "04-metadata-label-saved"
    Then 标签元数据应展示已保存配置
    When 我保存有效注解元数据配置
    And 截图 "05-metadata-annotation-saved"
    Then 注解元数据应展示已保存配置
