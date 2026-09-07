@TC-01 @P0 @deploy-flow @stateful
Feature: TC-01 部署应用
  作为运维人员
  我需要在 BKMS 平台的部署管理页对应用执行立即部署
  以验证应用部署功能端到端可用

  Background:
    Given AccessToken 认证已配置

  @space:default @app:default
  Scenario: 未部署空态应提示立即部署
    Given 我在当前应用的部署管理页
    And 当前应用已部署则跳过本用例
    Then 应用应处于未部署状态
    And 截图 "01-initial-empty-state"

  @space:default @app:default
  Scenario: 立即部署单实例并等待 Pod 就绪
    Given 我在当前应用的部署管理页
    And 当前应用已部署则跳过本用例
    When 我立即部署 1 个实例
    And 截图 "02-deploy-submitted"
    Then 实例列表应出现 1 个 Running 且 Healthy 的 Pod，最多等待 180 秒
    And 截图 "03-instance-ready"
