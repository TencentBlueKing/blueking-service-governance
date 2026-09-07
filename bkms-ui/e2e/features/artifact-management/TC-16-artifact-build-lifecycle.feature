@TC-16 @P1 @deploy-flow @stateful
Feature: TC-16 构建产物生命周期
  作为运维人员
  我需要同步、晋级、删除构建产物并查看 Helm Chart 构建记录
  以确认构建产物与部署前置链路可用

  Background:
    Given AccessToken 认证已配置

  @space:default @app:trpc @appType:trpc
  Scenario: 容器镜像支持一键同步
    Given 容器镜像接口返回生命周期测试数据
    Given 我在当前应用的制品管理页
    When 我执行容器镜像一键同步
    Then 容器镜像同步结果应展示
    And 截图 "01-container-image-sync"

  @space:default @app:trpc @appType:trpc
  Scenario: 容器镜像支持晋级
    Given 容器镜像接口返回生命周期测试数据
    Given 我在当前应用的制品管理页
    When 我晋级首个容器镜像 Tag
    Then 容器镜像应展示已晋级状态
    And 截图 "02-container-image-promoted"

  @space:default @app:trpc @appType:trpc
  Scenario: 容器镜像支持删除
    Given 容器镜像接口返回生命周期测试数据
    Given 我在当前应用的制品管理页
    When 我删除首个容器镜像 Tag
    Then 容器镜像 Tag 应从列表移除
    And 截图 "03-container-image-deleted"

  @space:default @app:trpc @appType:trpc
  Scenario: 容器镜像删除权限异常应展示配置引导
    Given 容器镜像接口返回删除权限异常测试数据
    Given 我在当前应用的制品管理页
    When 我删除首个容器镜像 Tag
    Then 容器镜像删除权限错误应展示构建配置引导
    And 截图 "04-container-image-delete-permission"

  @space:default @app:helm @appType:helm
  Scenario: Helm Chart 支持搜索详情和新建版本构建记录
    Given Helm Chart 接口返回生命周期测试数据
    Given 我在 Helm 应用的 Helm Chart 制品页
    When 我按约定版本搜索 Helm Chart
    Then Helm Chart 搜索结果应匹配版本
    When 我打开 Helm Chart 版本详情
    Then Helm Chart 版本详情应展示文件内容
    When 我新建 Helm Chart 版本构建
    Then Helm Chart 构建记录侧栏应展示目标构建
    And 截图 "05-helm-chart-build-records"
