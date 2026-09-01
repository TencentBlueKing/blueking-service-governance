@TC-04 @P0 @config-flow @stateful
Feature: TC-04 修改部署资源规格
  作为运维人员
  我需要在应用配置中修改部署的资源规格并能恢复默认
  以验证资源规格自定义与回滚能力

  Background:
    Given AccessToken 认证已配置

  @space:default @app:default
  Scenario: 修改测试环境的资源规格后保存
    Given 我在当前应用的测试环境部署配置页
    And 截图 "01-resource-view-mode"
    When 我修改资源规格
      | CPU预留 | 0.5   |
      | CPU限制 | 1     |
      | 内存预留 | 512Mi |
      | 内存限制 | 1Gi   |
    And 截图 "02-resource-saved"
    Then 资源规格应处于查看态
    And 资源规格区域应包含 "0.5"
