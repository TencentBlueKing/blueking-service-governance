# 版本历史

## 1.0.0-alpha.29

- remove rabbitmq dependency, async tasks are fully served by asynq
  （不兼容变更：`externalRabbitMQ` 已移除，升级前请从 values 中清理该段配置）

## 1.0.0-alpha.5

- update bkms server rabbitmq arguments

## 1.0.0-alpha.4

- bkms server support rabbitmq

## 1.0.0-alpha.3

- add trafficManager client config

## 1.0.0-alpha.2

- add external redis configuration

## 1.0.0-alpha.1

- bkms helm apiserver deployment support
