# Cassem Key 存储问题清单

本文档记录当前 key 设计与实际读写流程中已发现的问题，供逐项评审与取舍。内容基于源码静态审查，不代表已经完成修复。

## 评审状态说明

| 状态 | 含义 |
|---|---|
| 待决策 | 需要决定是否修、怎么修、何时修 |
| 建议修复 | 风险明确，建议进入修复计划 |
| 已决策 | 已确定修复方向，等待实现或排期 |
| 可观察 | 可先补监控/巡检，再决定是否改设计 |
| 已确认非问题 | 相关路径已经有保护，不作为问题处理 |

## 总览

| ID | 问题 | 严重级别 | 建议状态 | 主要影响 |
|---|---|---|---|---|
| KS-001 | instance reverse key 用 `-` 拼接，存在碰撞 | P0 | 已决策 | 改用 `@@@` 分隔符，避免现有字符集内碰撞 |
| KS-002 | 删除 app/env/element 不清理 operation history | P0 | 已决策 | 删除对象时同步删除对应 operation prefix |
| KS-003 | 元素创建/更新/发布是多次独立写入，缺少事务 | P1 | 已决策 / 延迟修复 | 方向为事务化，具体实现待定 |
| KS-004 | TTL 是懒清理，物理 key 可能长期残留 | P1 | 已决策 / 延迟修复 | 方向为类 etcd Lease TTL 模型 |
| KS-005 | instance normalized/reversed 索引写删不同步 | P1 | 已决策 | 短期返回 partial error，长期依赖 KS-003 事务根治 |
| KS-006 | key 多后，query 与 retention GC 可能退化 | P2 | 已决策 / 延迟修复 | query/retention 性能问题延迟处理 |
| KS-007 | `GetKVs` 静默吞掉单 key 错误 | P2 | 已决策 | 增加 per-key error，由调用方判断 NotFound 等错误 |
| KS-008 | elements 子树目录删除本身是递归删除 | Info | 已确认非问题 | `metadata` 与 `vN` 能随 element/env/app 删除 |

---

## KS-001：instance reverse key 用 `-` 拼接，存在碰撞

**状态：已决策**

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

### 决策

保留 reverse index，仍使用单段组合 key，但分隔符从 `-` 改为 `@@@`：

```text
cassem/instances/reversed/{app}@@@{env}@@@{key}/{instanceId}
```

选择 `@@@` 的原因：

- 当前 `app/env/key` 正则只允许字母、数字、`_`、`-`，不允许 `@`。
- 在当前字段约束下，`@@@` 不会和业务值碰撞。
- 改动小于层级 key，不需要调整 reverse prefix 层级语义。
- 可读性好于 length-prefix 或 escaping。

### 明确不做

- 不做迁移兼容。
- 不双写旧 key。
- 不读新 key fallback 旧 key。
- 不为旧 `cassem/instances/reversed/{app}-{env}-{key}` 做专项清理。

### 建议动作

- 增加 reverse key 分隔符常量，例如 `_INS_REVERSED_SEP = "@@@"`。
- `GenInstanceReversedKey` 改为使用 `app + "@@@" + env + "@@@" + key`。
- `GenInstanceReversedKeyWithInsId` 同步使用新分隔符。
- 增加 key generator 单元测试，覆盖含 `-` 的 app/env/key 不碰撞。
- 保持 `app/env/key` validate 规则不允许 `@`；如果未来允许 `@`，必须重新评审该分隔符。

---

## KS-002：删除 app/env/element 不清理 operation history

**状态：已决策**

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

### 决策

删除 element/env/app 时，同步删除对应 operation prefix：

```text
DeleteElement      => cassem/operations/{app}/{env}/{key}
DeleteEnvironment  => cassem/operations/{app}/{env}
DeleteApp          => cassem/operations/{app}
```

选择该方案的原因：

- 删除语义更直接：业务对象删除后，对应操作历史也删除。
- 不引入 operation-only GC。
- 不引入 tombstone。
- 不考虑历史迁移与兼容。

### 明确不做

- 不保留删除审计 tombstone。
- 不实现独立 operation-only GC。
- 不保留已删除对象的完整 operation history。

### 建议动作

- 增加 operation prefix key generator：app/env/element 三个层级。
- `DeleteElement` 删除 element prefix 后，删除 `cassem/operations/{app}/{env}/{key}`。
- `DeleteEnvironment` 删除 env elements prefix 后，删除 `cassem/operations/{app}/{env}`。
- `DeleteApp` 删除 app elements prefix 后，删除 `cassem/operations/{app}`。
- 调整或移除依赖已删除对象 operation history 的测试预期。

---

## KS-003：元素创建/更新/发布是多次独立写入，缺少事务

**状态：已决策 / 延迟修复**

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

### 决策

方向确定为事务化修复，但延迟处理；具体实现待定。

后续实现需要重新设计并评审：

- API 形态：`BatchKV`、`TxnKV`，或两者结合。
- 事务能力：仅原子批量写，还是支持 compare/CAS。
- Raft command 结构：一条 log 内承载多 put/delete/compare。
- bbolt apply 语义：同一个 update transaction 内完成全部操作。
- 失败语义：任一 op 失败时整个事务失败，不产生部分写入。

### 明确不做

- 当前阶段不实现业务层补偿。
- 当前阶段不引入只靠巡检修复的 eventual consistency 方案。
- 当前阶段不决定事务 API 细节。

### 建议动作

- 单独建立事务化设计任务。
- 先写 API/raft/storage 设计文档，再动代码。
- 设计时覆盖 create/update/publish/delete/app deletion/instance index 写入场景。
- 保留一致性巡检作为辅助工具，不作为替代方案。

---

## KS-004：TTL 是懒清理，物理 key 可能长期残留

**状态：已决策 / 延迟修复**

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

### 决策

采用类 etcd Lease TTL 模型作为修复方向，但延迟实现。

Lease 模型语义：

- 服务端 grant lease，返回 lease ID 与 TTL。
- key 写入时 attach 到 lease，不再只依赖每个 key 自身的懒过期判断。
- 客户端通过 keepalive 续约 lease。
- lease 过期后，服务端通过 Raft 共识提交 revoke/delete，主动删除该 lease 关联的所有 key。

Cassem 预期映射：

- 一个 instance 使用一个 instance lease。
- `cassem/instances/normalized/{instanceId}` 与该 instance 的所有 `cassem/instances/reversed/.../{instanceId}` 共享同一个 instance lease。
- 一个 agent 使用一个 agent lease。
- `cassem/agents/{agentId}` 绑定到对应 agent lease。

该方向用于解决 TTL lazy cleanup 导致的物理 key 长期残留问题，降低过期 instance/agent key 长期进入后续 snapshot 的概率。

具体 API、Raft command、storage schema、lease checkpoint/recovery、keepalive 行为与失败语义后续单独设计。

### 明确不做

- 当前阶段不实现代码改动。
- 当前阶段不实现针对 `cassem/instances` / `cassem/agents` 的定向 sweeper。
- 当前阶段不实现通用 per-key TTL index。
- 当前文档决策不考虑历史迁移与兼容。
- 不把 lazy cleanup 作为最终修复方案；可保留为防御性兜底，但不是主路径。
- lease 只解决过期清理一致性，不替代 KS-003/KS-005 中多 key 写入原子性问题。

### 建议动作

- 将 KS-004 后续任务改为 lease 机制设计任务，而不是 sweeper / TTL index 实现任务。
- 设计时先定义 lease API 与生命周期：grant、attach、keepalive、revoke、expire。
- 设计时覆盖 instance 场景：normalized key 与 reversed keys 必须绑定同一个 lease，避免只清理一侧索引。
- 设计时覆盖 agent 场景：agent key 绑定 agent lease，续约失败或过期后主动删除。
- 设计时明确 lease 过期删除必须通过 Raft 共识提交，保证各节点状态一致。
- 设计时明确恢复语义：节点重启、leader 切换、snapshot restore 后，lease 剩余 TTL 与已绑定 key 的处理方式必须可恢复。

---

## KS-005：instance normalized/reversed 索引写删不同步

**状态：已决策**

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

### 决策

短期采用 partial error，修正错误语义；长期依赖 KS-003 事务化能力根治。

`RegisterInstance` / `RenewInstance` 仍按现有顺序写入 normalized key 与多个 reversed keys，但 reversed 写入失败不能再被静默视为成功：

- normalized 写入失败：直接返回错误。
- 任一 reversed key 写入失败：返回错误，调用方可重试。
- 当前阶段不做复杂 rollback，不声明 normalized + reversed 写入具备原子性。

`UnregisterInstance` 仍按现有顺序删除 normalized key 与多个 reversed keys，但 reversed 删除失败必须显式返回：

- normalized 删除失败：直接返回错误。
- reversed 删除继续尽力处理所有 key。
- 收集所有 reversed delete 失败并返回聚合错误，避免被后续循环覆盖。

长期根治方案依赖 KS-003 事务化能力：normalized 与 reversed 的写入/删除应在同一个事务内完成，任一子操作失败时整个事务失败，不产生部分索引状态。

### 明确不做

- 当前阶段不实现 normalized/reversed 的复杂 rollback 或补偿事务。
- 当前阶段不声明 instance index 写入/删除具备原子性。
- 当前阶段不引入 migration/compat 逻辑。
- 不把 KS-004 lease 模型视为本问题的原子性修复；lease 只覆盖过期清理，不解决写入/删除半成功。

### 建议动作

- `setInstanceInfo` 中 reversed write 失败时返回错误，不再只记录日志后返回 `nil`。
- `RegisterInstance` / `RenewInstance` 在任一 reversed key 写入失败时向调用方返回错误。
- `UnregisterInstance` 删除 reversed keys 时记录所有失败 key。
- `UnregisterInstance` 完成全部 reversed delete 尝试后，如存在失败，返回包含失败 key 列表的聚合错误。
- 为 register/renew/unregister 增加测试，覆盖 reversed write/delete 失败不会被静默吞掉。
- 在 KS-003 事务设计中覆盖 instance normalized + reversed index 的 write/delete 场景。

---

## KS-006：key 多后，query 与 retention GC 可能退化

**状态：已决策 / 延迟修复**

### 现状

底层 `Range` 使用 bbolt cursor，并支持 `seek/limit`，普通分页不是全量扫描。

业务层 substring query 也会传入 `seek` 并使用 `nextSeek` 推进分页；但过滤条件是 `strings.Contains`，无法利用 key 排序直接定位命中项，因此低命中时仍可能不断分页扫描，直到凑够结果或扫完整个 env。

Retention GC 默认每 10 分钟最多处理 20 个元素。

### 风险

- 大 env 下低命中 query 会扫大量 key。
- query 分页语义可能不够精确：当前 query 返回的 `NextSeek` 来自第 `limit+1` 个匹配 key，而不是底层扫描 cursor。
- key 总量大时，retention 完整扫描周期很长。
- 写入速度高于清理速度时，版本与 operation backlog 增长。

### 证据

- bbolt Range 使用 cursor：`internal/cassemdb/infras/storage/bbolt_impl.go:249`
- Range limit 循环：`internal/cassemdb/infras/storage/bbolt_impl.go:259`
- 普通列表传入 `seek`：`internal/coordinator/coordinator_kv_r.go:132`
- query 分支用 `nextSeek := seek`：`internal/coordinator/coordinator_kv_r.go:159`
- query 分支 Range 传入 `Seek: nextSeek`：`internal/coordinator/coordinator_kv_r.go:161`
- query 分支做 substring filter：`internal/coordinator/coordinator_kv_r.go:172`
- query 未凑够结果会继续 Range：`internal/coordinator/coordinator_kv_r.go:160`
- query 返回 `NextSeek: matched[limit]`：`internal/coordinator/coordinator_kv_r.go:200`
- retention 默认 interval 10m：`pkg/conf/cassemadm.go:34`
- retention 默认每轮 20 个元素：`pkg/conf/cassemadm.go:35`
- retention 每轮按 `maxElementsPerRun` 停止：`internal/cassemadm/app/retention_gc.go:72`

### 决策

该问题延迟修复。

当前不限制 query 语义，不新增搜索索引，不调整 retention 吞吐策略。后续需要单独评审 query 分页语义、contains 搜索是否保留、是否改为 prefix search、是否增加搜索索引，以及 retention 指标和默认吞吐配置。

### 明确不做

- 当前阶段不改 query 行为。
- 当前阶段不把 contains query 改为 prefix search。
- 当前阶段不增加搜索索引。
- 当前阶段不调整 retention 默认值或新增 retention 指标。

### 建议动作

- 单独建立 query/retention 性能优化任务。
- 后续先确认 contains 搜索是否为必须能力。
- 如果保留 contains 搜索，优先评审最大扫描页数 / 最大扫描 key 数限制。
- 如果不保留 contains 搜索，改为 prefix search 以利用 bbolt key 排序。
- 后续评审 retention backlog、scan duration、deleted versions/operations 等指标。

---

## KS-007：`GetKVs` 静默吞掉单 key 错误

**状态：已决策**

### 现状

`GetKVs` 对每个 key 调用 `getKV`，失败时直接 `continue`。

### 风险

- 部分 key 缺失时，调用方只看到结果变少，不知道哪些 key 缺失。
- `NotFound` 不一定是请求级失败，但当前调用方无法区分缺失 key 与其它读取错误。
- metadata/version 不一致可能被隐藏。
- 排查 partial write 更困难。

### 证据

- `GetKVs` 循环：`internal/cassemdb/app/app_grpc.go:53`
- 单 key 错误直接跳过：`internal/cassemdb/app/app_grpc.go:56`
- 成功项才 append：`internal/cassemdb/app/app_grpc.go:61`

### 决策

扩展 `GetKVs` 响应，增加 per-key error 信息；保留部分成功语义，不把任意单 key 失败强制提升为整个请求失败。

建议响应形态：

```proto
message getKVsResp {
  repeated entity entities = 1;
  repeated keyError errors = 2;
}

message keyError {
  string key = 1;
  string code = 2;
  string message = 3;
}
```

语义约定：

- `entities` 只包含成功读取到的 key。
- `errors` 记录失败或缺失的 key。
- `NotFound` 作为 per-key error 返回，由调用方决定是忽略、降级还是视为失败。
- 非 `NotFound` 错误也进入 `errors`，并保留可判别的错误 code/details，便于调用方区分数据缺失与真实读失败。
- `GetKVs` 请求本身只在请求级错误时失败，例如参数非法、服务不可用、权限失败等。

### 明确不做

- 不继续静默吞掉单 key 错误。
- 不把 `NotFound` 固定定义为整个 `GetKVs` 的失败。
- 不将 strict whole-request failure 作为主方向。
- 不新增 `GetKVsStrict` 作为主要方案。
- 不考虑旧响应兼容或迁移成本。

### 建议动作

- 在 proto 中为 `GetKVs` response 增加 `repeated keyError errors`。
- 定义 `keyError`，至少包含 `key`、错误 `code` 与错误详情/message。
- 修改 `internal/cassemdb/app/app_grpc.go` 中 `GetKVs` 循环：成功 append 到 `entities`，失败 append 到 `errors`。
- 保持批量读取的 partial success：单 key 失败不直接中断其它 key 读取。
- 调整 coordinator/调用方逻辑：不要仅依赖返回数量判断结果，改为显式检查 `errors` 并自行决定 `NotFound` 是否可接受。
- 增加测试覆盖：全部成功、部分 `NotFound`、部分非 `NotFound` 错误、成功与错误混合返回。

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

1. **reverse key**：已决策，保留 reverse index，分隔符改为 `@@@`。
2. **operation history**：已决策，删除对象时同步删除对应 operation prefix。
3. **事务化**：已决策方向，延迟修复，具体实现待定。
4. **TTL 主动清理**：已决策方向，采用类 etcd Lease TTL 模型，延迟实现。
5. **instance 索引一致性**：已决策，短期返回 partial error，长期依赖事务化。
6. **query 与 retention 吞吐**：已决策，延迟修复。
7. **GetKVs 错误语义**：已决策，增加 per-key error，由调用方判断。

## 后续可拆任务

| 任务 | 前置决策 |
|---|---|
| 修改 reverse key 分隔符为 `@@@` | KS-001 |
| 删除对象时同步删除 operation prefix | KS-002 |
| 设计事务化 KV 写入能力 | KS-003 |
| 设计类 etcd Lease TTL 机制 | KS-004 |
| 修正 instance reversed 写删错误语义 | KS-005 |
| 评审 query/retention 性能优化 | KS-006 |
| 为 `GetKVs` 增加 per-key errors | KS-007 |
