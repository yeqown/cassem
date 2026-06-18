# Cassem Key 存储结构

Cassem 使用 `cassem/` 作为统一 KV 根前缀。key 由 `api/concept/key_generator.go` 生成，实际 value 多为 protobuf 序列化数据，通过 cassemdb 写入底层 BoltDB。

## 总览

| 逻辑域      | Key 结构                                                        | Value 类型                   | 作用                                  |
|----------|---------------------------------------------------------------|----------------------------|-------------------------------------|
| 配置元数据    | `cassem/elements/{app}/{env}/{key}/metadata`                  | `concept.ElementMetadata`  | 保存配置 key、app、env、版本指针、内容类型、当前使用版本指纹 |
| 配置版本     | `cassem/elements/{app}/{env}/{key}/v{N}`                      | `concept.Element`          | 保存第 `N` 个版本原始内容与发布状态                |
| 应用元数据    | `cassem/apps/{appId}`                                         | `concept.AppMetadata`      | 保存应用描述、负责人、状态、访问 secret             |
| 环境目录     | `cassem/elements/{app}/{env}`                                 | 目录 key                     | 给应用环境建立逻辑前缀，供列表与删除使用                |
| 操作记录     | `cassem/operations/{app}/{env}/{key}/operations/{operatedAt}` | `concept.ElementOperation` | 保存配置变更审计记录                          |
| 实例正向索引   | `cassem/instances/normalized/{instanceId}`                    | `concept.Instance`         | 保存客户端实例完整信息，TTL 120 秒               |
| 实例反向索引   | `cassem/instances/reversed/{app}@@@{env}@@@{key}/{instanceId}`    | `instanceId` bytes         | 从配置项反查正在 watch 该配置的实例，TTL 120 秒     |
| Agent 节点 | `cassem/agents/{agentId}`                                     | `concept.AgentInstance`    | 保存 agent 注册信息，TTL 由调用方传入            |
| ACL 策略   | `cassem/acl/policy`                                           | `concept.Casbin`           | 保存 Casbin policy/grouping policy    |
| 用户       | `cassem/acl/users/{account}`                                  | `concept.User`             | 保存用户账号、昵称、salt、hash 后密码、状态          |

## 配置元素 namespace

配置元素使用三层业务维度定位：`app`、`env`、`key`。

```text
cassem/elements/{app}/{env}/{key}/metadata
cassem/elements/{app}/{env}/{key}/v1
cassem/elements/{app}/{env}/{key}/v2
...
```

示例：

```text
cassem/elements/demo/prod/db_url/metadata
cassem/elements/demo/prod/db_url/v1
cassem/elements/demo/prod/db_url/v2
```

字段约束来自 proto validate：

- `app`：`^[A-Za-z0-9][A-Za-z0-9_-]*$`
- `env`：`^[A-Za-z0-9][A-Za-z0-9_-]*$`
- `key`：`^[A-Za-z0-9][A-Za-z0-9_-]*$`

因此配置 key 不支持 `/` 分层；业务层级靠 app/env/key 三元组表达。

### Metadata

`metadata` 保存 `concept.ElementMetadata`：

- `key/app/env`：业务身份
- `latestVersion`：已创建最大版本号
- `unpublishedVersion`：当前未发布版本；非 0 时不能继续 update
- `usingVersion`：正在使用版本；未发布前可为 0
- `usingFingerprint`：当前使用版本 raw 内容 MD5
- `contentType`：JSON/TOML/INI/PLAINTEXT 等

### Version

`v{N}` 保存 `concept.Element`：

- `raw`：配置原始内容 bytes
- `version`：版本号，从 1 开始
- `published`：该版本是否已经发布
- `metadata`：读取聚合结果时附加，存储版本 value 本身主要保存 raw/version/published

### 写入生命周期

1. `CreateElement`：写 `metadata`，写 `v1`，`v1.published=false`，`unpublishedVersion=1`，记录 SET 操作。
2. `PublishElementVersion`：读取目标版本，更新 `metadata.usingVersion`、`usingFingerprint`，清空 `unpublishedVersion`，标记版本 `published=true`，记录 PUBLISH 操作。
3. `UpdateElement`：要求 `unpublishedVersion=0`，写新版本 `v{latest+1}`，更新 metadata，记录 SET 操作。
4. `RollbackElementVersion`：读取历史版本，更新 `usingVersion` 与指纹，记录 PUBLISH 操作，remark 写 rollback 信息。
5. `DeleteElement`：删除 `cassem/elements/{app}/{env}/{key}` 目录前缀，并删除对应 `cassem/operations/{app}/{env}/{key}` 操作记录前缀。

## 应用与环境

```text
cassem/apps/{appId}
cassem/elements/{app}
cassem/elements/{app}/{env}
```

- `cassem/apps/{appId}` 保存 `concept.AppMetadata`。
- `cassem/elements/{app}/{env}` 是环境目录 key，用于按 app/env range 列出配置项。
- 删除应用时先删除 `cassem/elements/{app}` 目录前缀，再删除 `cassem/apps/{appId}`。

## 操作记录

```text
cassem/operations/{app}/{env}/{key}/operations/{operatedAt}
```

`operatedAt` 使用 `time.Now().UnixNano()`，天然按时间递增排序。value 为 `concept.ElementOperation`：

- `operator`：操作人，来自 request context
- `operatedAt`：纳秒时间戳
- `operatedKey`：配置 key
- `op`：`SET` / `UNSET` / `PUBLISH`
- `lastVersion`：操作前版本
- `currentVersion`：操作后版本
- `remark`：备注，例如 rollback 信息

## 实例索引

客户端实例注册时写两类 key：

```text
cassem/instances/normalized/{instanceId}
cassem/instances/reversed/{app}@@@{env}@@@{key}/{instanceId}
```

`instanceId` 由 `Instance.Id()` 生成，正向索引 value 为完整 `concept.Instance`。反向索引 value 只保存 `instanceId` bytes，用于根据配置项查找所有 watch 该配置的实例。

两类实例 key 都设置 TTL 120 秒。`RenewInstance` 会覆盖刷新正向索引与反向索引；`UnregisterInstance` 会删除正向索引，并根据实例中 `watching` 列表删除对应反向索引。

## Agent 节点

```text
cassem/agents/{agentId}
```

value 为 `concept.AgentInstance`，包含：

- `agentId`
- `addr`
- `annotations`

注册时 `Overwrite=false`，续约时 `Overwrite=true`。TTL 由调用方传入。agent 变化可通过 watch `cassem/agents` 前缀感知。

## ACL 与用户

```text
cassem/acl/policy
cassem/acl/users/{account}
```

- `cassem/acl/policy` 保存 `concept.Casbin`，其中包含 Casbin `p` 与 `g` 策略行。
- `cassem/acl/users/{account}` 保存 `concept.User`。密码保存为带 salt hash 后值，不保存明文。

## Range 与分页语义

代码中常见 range 前缀：

| 查询                   | Range key                                                      |
|----------------------|----------------------------------------------------------------|
| 列出 app               | `cassem/apps`                                                  |
| 列出某 app 下 env/domain | `cassem/elements/{app}`                                        |
| 列出某 env 下元素          | `cassem/elements/{app}/{env}`                                  |
| 列出某元素版本              | `cassem/elements/{app}/{env}/{key}`，默认 seek 为 `v` 以跳过 metadata |
| 列出元素操作               | `cassem/operations/{app}/{env}/{key}/operations`               |
| 列出实例                 | `cassem/instances/normalized`                                  |
| 按元素反查实例              | `cassem/instances/reversed/{app}@@@{env}@@@{key}`                  |
| 列出 agent             | `cassem/agents`                                                |
| 列出用户                 | `cassem/acl/users`                                             |

分页由 cassemdb `Range` 返回 `hasMore` 与 `nextSeekKey`，调用方下一页使用 `seek=nextSeekKey`。

## 设计要点

- 配置内容与版本指针分离：`metadata` 存指针，`v{N}` 存内容。
- 发布前更新受控：存在 `unpublishedVersion` 时拒绝创建下一版本。
- watch 分发依赖反向索引：从配置项快速查到订阅实例。
- 临时在线状态使用 TTL：client instance 与 agent 注册信息会自动过期。
- ACL 独立存储：策略集中在一个 key，用户按 account 分 key 存储。
