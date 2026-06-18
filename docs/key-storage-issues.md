# Cassem Key 存储问题清单

本文档记录当前 key 设计与实际读写流程中已发现的问题，供逐项评审与取舍。内容基于源码静态审查，不代表已经完成修复。

## 评审状态说明

| 状态 | 含义 |
|---|---|
| 待决策 | 需要决定是否修、怎么修、何时修 |
| 建议修复 | 风险明确，建议进入修复计划 |
| 可观察 | 可先补监控/巡检，再决定是否改设计 |
| 已确认非问题 | 相关路径已经有保护，不作为问题处理 |

## 总览

| ID | 问题 | 严重级别 | 建议状态 | 主要影响 |
|---|---|---|---|---|
| KS-001 | instance reverse key 用 `-` 拼接，存在碰撞 | P0 | 建议修复 | 按配置项反查实例可能串数据 |
| KS-002 | 删除 app/env/element 不清理 operation history | P0 | 建议修复 | 已删除配置的审计记录长期残留 |
| KS-003 | 元素创建/更新/发布是多次独立写入，缺少事务 | P1 | 建议修复 | 半成功导致孤儿版本、元数据不一致、审计缺失 |
| KS-004 | TTL 是懒清理，物理 key 可能长期残留 | P1 | 可观察/建议修复 | instance/agent 过期后仍占存储和 snapshot |
| KS-005 | instance normalized/reversed 索引写删不同步 | P1 | 建议修复 | watch 反查漏实例或残留实例 |
| KS-006 | key 多后，query 与 retention GC 可能退化 | P2 | 可观察/建议优化 | 查询慢、历史清理追不上写入 |
| KS-007 | `GetKVs` 静默吞掉单 key 错误 | P2 | 建议修复 | 不一致被隐藏，排查困难 |
| KS-008 | elements 子树目录删除本身是递归删除 | Info | 已确认非问题 | `metadata` 与 `vN` 能随 element/env/app 删除 |

---

## KS-001：instance reverse key 用 `-` 拼接，存在碰撞

**状态：建议修复**

### 现状

反向索引用于从配置项反查正在 watch 该配置的实例：

```text
cassem/instances/reversed/{app}-{env}-{key}/{instanceId}
```

生成逻辑把 `app`、`env`、`key` 用 `-` 拼起来。

### 风险

`app/env/key` 都允许 `-`，因此不同三元组可能生成同一个 reverse prefix。

例子：

```text
app=a-b, env=c,   key=d  => a-b-c-d
app=a,   env=b-c, key=d  => a-b-c-d
```

影响：

- `GetInstancesByElement` 可能返回另一个配置项的实例。
- unregister 可能删除另一个配置项的 reverse index。
- watch/dispatch 相关管理能力可能串数据。

### 证据

- reverse key 拼接：`api/concept/key_generator.go:111`
- reverse key + instanceId 拼接：`api/concept/key_generator.go:116`
- `app/env/key` regex 允许 `-`：`api/concept/types.proto:38`、`api/concept/types.proto:101`
- 反查实例依赖 reverse key：`internal/coordinator/coordinator_ins_hybrid.go:89`
- 写 reverse key：`internal/coordinator/coordinator_ins_hybrid.go:211`
- 删除 reverse key：`internal/coordinator/coordinator_ins_hybrid.go:284`

### 决策选项

1. **推荐：保留 reverse index，改成层级 key**

   ```text
   cassem/instances/reversed/{app}/{env}/{key}/{instanceId}
   ```

   优点：无 delimiter 碰撞；支持按 app/env/key 层级清理。

2. **保留现结构，但做 escaping/length-prefix**

   例如：`{len(app)}:{app}{len(env)}:{env}{len(key)}:{key}`。可避免碰撞，但可读性差。

3. **删除 reverse index，改扫 normalized**

   可行但会退化为全量实例扫描。只适合实例量很小或不需要按配置项反查实例。

### 建议动作

- 新增 `GenInstanceReversedDirKey(app, env, key)` 层级结构。
- 迁移期双写新旧 key。
- 读路径先读新 key，空结果 fallback 旧 key。
- unregister 双删新旧 key。
- 后台清理旧 `cassem/instances/reversed/{app}-{env}-{key}`。

---

## KS-002：删除 app/env/element 不清理 operation history

**状态：建议修复**

### 现状

配置元素存在 `elements` 树中：

```text
cassem/elements/{app}/{env}/{key}/metadata
cassem/elements/{app}/{env}/{key}/v{N}
```

操作历史存在独立 `operations` 树中：

```text
cassem/operations/{app}/{env}/{key}/operations/{operatedAt}
```

删除元素、环境、应用时，只删除 `elements` 树或 app metadata，不删除对应 `operations` 树。

### 风险

- 已删除元素的操作历史长期残留。
- 已删除 env/app 下所有元素历史仍可能保留。
- retention compaction 依赖现存 element metadata；metadata 删除后，旧 operation 更难被发现并清理。

### 证据

- operation key 生成：`api/concept/key_generator.go:68`
- operation 写入：`internal/coordinator/coordinator_kv_w.go:296`
- `DeleteElement` 删除 elements 目录后写 UNSET operation：`internal/coordinator/coordinator_kv_w.go:109`
- `DeleteEnvironment` 只删除 env elements 目录：`internal/coordinator/coordinator_kv_w.go:140`
- `DeleteApp` 删除 app elements 目录与 app metadata：`internal/coordinator/coordinator_kv_w.go:242`
- compaction 先加载 element metadata：`internal/cassemdb/app/retention_compact.go:45`
- compaction 按 metadata 定位 operations：`internal/cassemdb/app/retention_compact.go:55`
- retention GC 从 `cassem/elements` 扫现存元素：`internal/cassemadm/app/retention_gc.go:154`

### 决策选项

1. **推荐：增加独立 operation GC**

   直接扫描 `cassem/operations`，按 `operatedAt` 保留 N 天，不依赖 element metadata。

2. **删除元素时同步删除 operation 前缀**

   `DeleteElement` 删除：

   ```text
   cassem/operations/{app}/{env}/{key}
   ```

   缺点：如果审计要求保留删除历史，会丢失历史。

3. **保留审计，但写 tombstone**

   删除元素后保留 operation 一段时间，记录 tombstone，GC 到期后清理。

### 建议动作

- 明确审计保留策略：删除后保留多久。
- 实现 operation-only GC。
- 管理端增加 operation backlog 指标。

---

## KS-003：元素创建/更新/发布是多次独立写入，缺少事务

**状态：建议修复**

### 现状

一个业务操作通常会写多个 key，但每个 key 都通过独立 `SetKV`/`UnsetKV` 发起独立 Raft propose。

### 风险

任意一步失败都会产生半成功状态。

典型场景：

- `CreateElement`：metadata 成功、`v1` 失败 => metadata 指向不存在版本。
- `UpdateElement`：`vN` 成功、metadata 失败 => 孤儿版本。
- `PublishElementVersion`：metadata 成功、version `published=true` 失败 => metadata 使用版本与版本状态不一致。
- operation 写失败 => 数据变化存在，但审计缺失。
- `DeleteApp`：elements 删除成功、app metadata 删除失败 => app 还在，配置没了。

### 证据

- `CreateElement` 写 metadata：`internal/coordinator/coordinator_kv_w.go:53`
- `CreateElement` 写 `v1`：`internal/coordinator/coordinator_kv_w.go:63`
- `CreateElement` 写 operation：`internal/coordinator/coordinator_kv_w.go:67`
- `UpdateElement` 写新版本：`internal/coordinator/coordinator_kv_w.go:97`
- `UpdateElement` 更新 metadata：`internal/coordinator/coordinator_kv_w.go:102`
- `PublishElementVersion` 更新 metadata：`internal/coordinator/coordinator_kv_w.go:220`
- `PublishElementVersion` 更新 version flag：`internal/coordinator/coordinator_kv_w.go:226`
- `DeleteApp` 分两次删除：`internal/coordinator/coordinator_kv_w.go:246`、`internal/coordinator/coordinator_kv_w.go:253`
- 单次 `SetKV` 是单条 Raft command：`internal/cassemdb/infras/raftimpl/etcdio/raft_impl.go:296`
- bbolt 单次 `SetKV` 只写一个 key 或目录：`internal/cassemdb/infras/storage/bbolt_impl.go:180`

### 决策选项

1. **推荐：casemdb 增加 Batch/Txn command**

   一条 Raft log 内包含多 put/delete/CAS 条件，apply 时在一个 bbolt transaction 内完成。

2. **业务层补偿**

   失败后尝试回滚已写 key。实现简单，但分布式失败场景仍复杂。

3. **接受 eventual consistency，增加巡检修复**

   定期扫描：metadata 指向版本是否存在、版本是否有 metadata、operation 是否缺失。

### 建议动作

- 新增 `BatchKV`/`TxnKV` API。
- 支持 CAS 条件：metadata `latestVersion/unpublishedVersion` 必须匹配。
- create/update/publish/delete 迁移到 batch。
- 增加一致性巡检，先发现历史脏数据。

---

## KS-004：TTL 是懒清理，物理 key 可能长期残留

**状态：可观察 / 建议修复**

### 现状

TTL 存在 entity 中。过期判断发生在读取或 range 时；物理删除也是访问时触发。

### 风险

- instance/agent 过期后，API 可能看不到，但 BoltDB 中仍可能存在。
- 如果没有后续访问该 prefix，过期 key 长期保留。
- snapshot 可能携带过期 key。
- Range 触发的过期删除是异步且忽略错误。

### 证据

- TTL 计算与过期判断：`internal/cassemdb/api/cassemdb.raft.pb.supplement.go:64`
- `GetKV` 发现过期后删除：`internal/cassemdb/infras/raftimpl/etcdio/raft_impl.go:414`
- `probeRemoveExpired` 调用 `UnsetKV`：`internal/cassemdb/infras/raftimpl/etcdio/raft_impl.go:421`
- `Range` 收集 expired keys：`internal/cassemdb/infras/storage/bbolt_impl.go:265`
- `Range` 异步删除 expired keys：`internal/cassemdb/infras/raftimpl/etcdio/raft_impl.go:458`
- 异步删除忽略错误：`internal/cassemdb/infras/raftimpl/etcdio/raft_impl.go:467`
- instance normalized TTL 120：`internal/coordinator/coordinator_ins_hybrid.go:200`
- instance reversed TTL 120：`internal/coordinator/coordinator_ins_hybrid.go:215`
- agent TTL 由调用方传入：`internal/coordinator/coordinator_agent_hybrid.go:93`

### 决策选项

1. **推荐：增加 TTL index + 后台 sweeper**

   ```text
   cassem/ttl/{expireAt}/{keyHash}
   ```

   到期主动删除目标 key。

2. **针对 instance/agent 做定向 sweeper**

   定期 range：

   ```text
   cassem/instances
   cassem/agents
   ```

3. **先加观测**

   暴露 expired keys count、cleanup failure、oldest expired key age。

### 建议动作

- 短期：记录 Range 异步删除失败。
- 中期：增加 instance/agent sweeper。
- 长期：增加通用 TTL index。

---

## KS-005：instance normalized/reversed 索引写删不同步

**状态：建议修复**

### 现状

注册/续约实例时先写 normalized，再写多个 reversed。reverse 写失败只记录日志，不返回错误。

注销实例时先删 normalized，再删 reversed。reverse 删除失败也只记录日志，且循环中 `err` 可能被后续操作覆盖。

### 风险

- normalized 存在，但 reversed 缺失 => 按配置反查实例漏数据。
- normalized 删除成功，reversed 删除失败 => 反查时得到 stale instanceId。
- reverse 写删不完整时只能等 TTL 或下次 renew 修复。

### 证据

- 写 normalized：`internal/coordinator/coordinator_ins_hybrid.go:190`
- 写 reversed：`internal/coordinator/coordinator_ins_hybrid.go:211`
- reverse 写失败只打日志：`internal/coordinator/coordinator_ins_hybrid.go:222`
- `setInstanceInfo` 最终返回 nil：`internal/coordinator/coordinator_ins_hybrid.go:233`
- unregister 先删 normalized：`internal/coordinator/coordinator_ins_hybrid.go:278`
- unregister 再删 reversed：`internal/coordinator/coordinator_ins_hybrid.go:284`
- reverse 删除失败只打日志：`internal/coordinator/coordinator_ins_hybrid.go:291`

### 决策选项

1. **推荐：把 normalized + reversed 写入放进 Batch/Txn**

   与 KS-003 一起解决。

2. **返回 partial error**

   reverse 写/删失败时返回聚合错误，调用方可重试。

3. **增加 reconcile job**

   定期扫描 normalized 与 reversed，修复不一致。

### 建议动作

- reverse 写失败不要静默成功。
- unregister 记录所有失败 key。
- 增加 `CheckInstanceIndexConsistency` 工具或后台任务。

---

## KS-006：key 多后，query 与 retention GC 可能退化

**状态：可观察 / 建议优化**

### 现状

底层 `Range` 使用 bbolt cursor，并支持 `seek/limit`，普通分页不是全量扫描。

但业务层 substring query 会不断分页扫描，直到凑够结果或扫完整个 env。

Retention GC 默认每 10 分钟最多处理 20 个元素。

### 风险

- 大 env 下低命中 query 会扫大量 key。
- key 总量大时，retention 完整扫描周期很长。
- 写入速度高于清理速度时，版本与 operation backlog 增长。

### 证据

- bbolt Range 使用 cursor：`internal/cassemdb/infras/storage/bbolt_impl.go:249`
- Range limit 循环：`internal/cassemdb/infras/storage/bbolt_impl.go:259`
- query 分支做 substring filter：`internal/coordinator/coordinator_kv_r.go:157`
- query 未凑够结果会继续 Range：`internal/coordinator/coordinator_kv_r.go:160`
- retention 默认 interval 10m：`pkg/conf/cassemadm.go:34`
- retention 默认每轮 20 个元素：`pkg/conf/cassemadm.go:35`
- retention 每轮按 `maxElementsPerRun` 停止：`internal/cassemadm/app/retention_gc.go:72`

### 决策选项

1. **限制 query 为 prefix search**

   利用 key 排序与 seek，避免 substring 全扫描。

2. **增加搜索索引**

   如果需要 contains 查询，维护 secondary index。

3. **提高 retention 吞吐**

   根据 keyspace 动态调整 `maxElementsPerRun` 与 interval。

4. **先加指标**

   暴露 retention backlog、scan duration、deleted versions/operations。

### 建议动作

- 中短期：加 retention backlog 指标，调整默认值。
- 长期：为查询建立索引或限制查询语义。

---

## KS-007：`GetKVs` 静默吞掉单 key 错误

**状态：建议修复**

### 现状

`GetKVs` 对每个 key 调用 `getKV`，失败时直接 `continue`。

### 风险

- 部分 key 缺失时，调用方只看到结果变少，不知道哪些 key 缺失。
- metadata/version 不一致可能被隐藏。
- 排查 partial write 更困难。

### 证据

- `GetKVs` 循环：`internal/cassemdb/app/app_grpc.go:53`
- 单 key 错误直接跳过：`internal/cassemdb/app/app_grpc.go:56`
- 成功项才 append：`internal/cassemdb/app/app_grpc.go:61`

### 决策选项

1. **推荐：响应中增加 per-key error**

   ```proto
   message getKVsResp {
     repeated entity entities = 1;
     repeated keyError errors = 2;
   }
   ```

2. **严格模式**

   任意 key 失败则整个 `GetKVs` 返回错误。

3. **保留兼容模式，新增 `GetKVsStrict`**

   避免破坏现有调用方。

### 建议动作

- 新增 strict API 或扩展 response。
- coordinator 调用时校验返回数量。

---

## KS-008：elements 子树目录删除本身是递归删除

**状态：已确认非问题**

### 说明

`DeleteElement`、`DeleteEnvironment`、`DeleteApp` 删除 `elements` 子树时，使用 `IsDir=true`。底层 bbolt 调用 `DeleteBucket`，会删除整个 bucket 子树。

因此 `cassem/elements/{app}/{env}/{key}/metadata` 与 `vN` 在目录删除成功时会一起删除。这里不是主要泄露点。

主要残留来自：

- `cassem/operations` 独立树。
- TTL key 懒清理。
- instance reverse 索引不一致。
- 多 key 写入半成功。

### 证据

- `DeleteElement` 使用 `IsDir=true`：`internal/coordinator/coordinator_kv_w.go:112`
- `DeleteEnvironment` 使用 `IsDir=true`：`internal/coordinator/coordinator_kv_w.go:142`
- `DeleteApp` 删除 app elements bucket：`internal/coordinator/coordinator_kv_w.go:246`
- bbolt 删除目录用 `DeleteBucket`：`internal/cassemdb/infras/storage/bbolt_impl.go:207`

---

## 建议决策顺序

1. **先定 reverse key 是否保留**
   - 建议保留，但改层级结构。

2. **再定 operation history 保留策略**
   - 删除后保留多久？是否必须审计可追溯？

3. **再定是否引入 Batch/Txn**
   - 影响 API、Raft command、storage apply，范围最大。

4. **再定 TTL 主动清理策略**
   - 通用 TTL index 还是先做 instance/agent 定向 sweeper。

5. **最后处理查询与 retention 吞吐**
   - 根据实际 key 数、写入频率、查询模式调优。

## 后续可拆任务

| 任务 | 前置决策 |
|---|---|
| 修改 reverse key 结构并迁移 | KS-001 |
| 实现 operation-only GC | KS-002 |
| 增加 Batch/Txn KV API | KS-003 |
| 增加 instance index reconcile | KS-005 |
| 增加 TTL sweeper | KS-004 |
| 增加 retention metrics | KS-006 |
| 增加 GetKVs strict/per-key errors | KS-007 |
