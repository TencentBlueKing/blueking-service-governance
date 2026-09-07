@TC-03 @P0 @deploy-flow @stateful
Feature: TC-03 移除部署
  作为运维人员
  我需要对已部署应用执行移除部署
  以验证 BKMS 的下线与清理能力

  Background:
    Given AccessToken 认证已配置

  @space:default @app:default
  Scenario: 移除部署后应回到未部署空态
    Given 我在当前应用的部署管理页
    And 当前应用未部署则先部署 1 个实例
    And 截图 "01-before-remove"
    Then 移除部署确认应要求输入环境名称
    When 我执行移除部署
    And 截图 "02-remove-submitted"
    Then 应用应处于未部署状态
    And 截图 "03-uninstalled-verified"
