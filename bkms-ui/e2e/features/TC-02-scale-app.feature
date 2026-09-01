@TC-02 @P0 @deploy-flow @stateful
Feature: TC-02 扩缩容
  作为运维人员
  我需要对已部署应用执行扩容和缩容
  以验证 BKMS 的实例调整能力

  Background:
    Given AccessToken 认证已配置

  @space:default @app:default
  Scenario: 手动调节扩容到 3 副本并等待就绪
    Given 我在当前应用的部署管理页
    And 当前应用未部署则先部署 1 个实例
    When 我使用手动调节扩缩容到 3 副本
    And 截图 "01-scale-up-submitted"
    Then 实例列表应出现 3 个 Running 且 Healthy 的 Pod，最多等待 180 秒
    And 截图 "02-3-replicas-ready"

  @space:default @app:default
  Scenario: 配置自动调节
    Given 我在当前应用的部署管理页
    And 当前应用未部署则先部署 1 个实例
    When 我配置自动调节最小 1 最大 3 CPU 使用率 80%
    And 截图 "03-auto-scale-configured"
    Then 自动调节配置应展示最小 1 最大 3 CPU 使用率 80%

  @space:default @app:default
  Scenario: 切回手动调节并缩容回 1 副本
    Given 我在当前应用的部署管理页
    And 当前应用未部署则先部署 1 个实例
    When 我使用手动调节扩缩容到 1 副本
    And 截图 "04-scale-down-submitted"
    Then 实例列表应出现 1 个 Running 且 Healthy 的 Pod，最多等待 180 秒
    And 截图 "05-1-replica-ready"
