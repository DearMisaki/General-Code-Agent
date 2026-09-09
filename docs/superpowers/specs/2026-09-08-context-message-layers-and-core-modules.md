# Context System Message Layers and Core Modules

## 设计目标

上下文系统的目标不是把更多文本塞进模型，而是把“哪些信息能进入模型、以什么层级进入、何时裁剪、如何交接、如何审计”变成一个明确的架构边界。

当前实现采用 `internal/contextmgr` 作为 Context Layer。它不直接替代 `conversation`、`compact`、`toolresult` 这些底层能力，而是把它们组织到一个统一的上下文准备流程里。Agent 主循环只需要在每次发起模型调用前经过 `ContextGateway.PrepareTurn`，由 Context Layer 负责构建快照、注入提示、做预算控制、生成审计记录，并返回最终发送给 LLM 的 conversation。

## 消息分层

Context Layer 把运行时信息分成六层：User、Session、Business、Execution、Environment、Budget。每一层都有不同来源、生命周期和暴露策略。

### User Layer

User Layer 表示来自用户长期意图和偏好的上下文。

它当前包含：

- `Instructions`：项目级或用户级指令。
- `MemoryContent`：长期记忆生成的上下文内容。

这层信息优先级最高，但不应该每次以普通消息反复追加。当前网关在进入构建流程前调用 `conversation.InjectLongTermMemory`，让长期指令和记忆以受控方式进入 conversation，避免重复注入。`Renderer` 仍然保留渲染能力，主要用于测试和后续兼容迁移。

设计边界是：User Layer 负责“用户长期要求是什么”，不负责当前会话中的工具结果、压缩摘要或执行状态。

### Session Layer

Session Layer 表示当前会话历史。

它当前包含：

- `SessionID`：会话标识。
- `Messages`：当前 conversation 中的消息快照。
- `MessageCount`：构建快照时的消息数量。

这层的关键要求是不能产生副作用。`Builder` 读取 `conversation.Manager.GetMessages()`，生成快照，但不向原 conversation 写入任何内容。这样可以让 Context Layer 在调试和审计时回答“准备本轮请求时，原始会话状态是什么”。

Session Layer 不决定压缩策略。压缩属于 Budget Layer，但压缩对象是 Session Layer 中的 conversation。

### Business Layer

Business Layer 表示任务能力和业务约束。

它当前包含：

- `ActiveSkills`：已经激活并 pin 到上下文里的技能 SOP。
- `ToolSchemas`：本轮可直接暴露给模型的工具 schema。
- `DeferredToolNames`：延迟加载的工具名称。
- `Checker`：权限和模式检查器，例如 Plan mode。

这层决定“模型现在可以做什么”和“哪些操作必须受限”。例如 Active Skills 会由 `Renderer` 渲染成系统提醒；Deferred Tools 会渲染成 ToolSearch 加载说明；Plan mode 会通过 `Checker` 触发 plan reminder。

Business Layer 不保存会话历史，也不决定 token 裁剪。它只表达能力、权限和流程约束。

### Execution Layer

Execution Layer 表示本轮 agent 执行状态。

它当前包含：

- `AgentID` 和 `AgentType`：当前执行者身份。
- `Protocol`：当前交互协议。
- `WorkDir`：工作目录。
- `Iteration` 和 `MaxIterations`：循环状态。
- `ContextWindow` 和 `MaxOutputTokens`：模型预算参数。

这层用于把“当前正在由哪个 agent、在哪个目录、以第几轮执行”这类元信息结构化。它主要服务于审计、调试、handoff 和预算控制。它通常不会直接完整渲染给模型，除非某个策略明确需要。

### Environment Layer

Environment Layer 表示运行环境。

它当前包含：

- 操作系统、架构、shell。
- 当前日期。
- Git 仓库状态和分支。
- 模型名称。

这层由 `prompt.DetectEnvironment` 提供基础信息，`Builder` 补齐 runtime fallback。它的作用是让上下文快照包含环境事实，便于后续做审计、恢复和跨 agent 交接。

Environment Layer 和 User Layer 的区别是：它描述“运行在哪里”，不是“用户想要什么”。

### Budget Layer

Budget Layer 表示上下文大小管理状态。

它当前包含：

- `UsageAnchor`：自动压缩的 token 使用锚点。
- `CompactTracking`：自动压缩追踪状态。
- `ReplacementState`：工具结果替换状态。

这层不表示业务语义，而表示“怎样在上下文窗口内安全地发送”。它负责连接 `compact` 和 `toolresult`：前者压缩旧 conversation，后者把过大的工具结果落盘并替换成轻量引用。

Budget Layer 的核心原则是：尽量保持模型可见内容足够完整，同时防止上下文窗口被工具结果或历史消息撑爆。

## 消息进入模型的路径

本轮请求进入 LLM 前经过以下路径：

1. Agent 主循环收集 conversation、工具 schema、权限状态、active skills、deferred tools、通知、预算状态和模型客户端。
2. `ContextGateway.PrepareTurn` 接收 `PrepareRequest`。
3. Gateway 先把长期指令和 memory 注入 conversation。
4. `Builder.Build` 读取当前输入，构建 `RuntimeContext` 快照。
5. `Renderer.Render` 根据快照和请求生成需要进入模型的系统提醒。
6. Gateway 把提醒追加到 conversation。
7. `Budgeter.PrepareBudget` 执行自动压缩和工具结果替换。
8. Gateway 生成审计记录。
9. Gateway 返回 `PreparedTurn`，其中 `APIConversation` 是最终发送给 LLM 的 conversation。

这个路径把“原始上下文是什么”和“最终发送给模型的上下文是什么”分开了。`Snapshot` 代表结构化观察结果，`APIConversation` 代表预算处理后的实际模型输入。

## 核心模块

### types.go

`types.go` 定义 Context Layer 的公共数据模型。

核心类型包括：

- `RuntimeContext`：一次上下文准备的结构化快照。
- `UserContext`、`SessionContext`、`BusinessContext`、`ExecutionContext`、`EnvironmentContext`、`BudgetContext`：六个消息层级。
- `PrepareRequest`：Agent 主循环传入 Context Layer 的边界对象。
- `PreparedTurn`：Context Layer 返回给 Agent 主循环的边界对象。
- `HandoffPackage`：跨 agent 交接包。
- `AuditRecord`：上下文事件审计记录。

这部分的设计重点是稳定边界。Agent、sub-agent、compact、toolresult 后续都可以继续演进，但它们和 Context Layer 的集成点应该尽量收敛到这些类型上。

### builder.go

`Builder` 负责从运行时输入构建 `RuntimeContext`。

输入是 `PrepareRequest`，输出是 `RuntimeContext`。它做的是只读采样：

- 从 conversation 拷贝消息快照。
- 拷贝 active skills。
- 拷贝 tool schemas 和 deferred tool names。
- 记录 agent、协议、工作目录、迭代次数和 token 参数。
- 读取环境信息。
- 记录 `ContextSource`，说明每类上下文来自哪里。

`Builder` 不向 conversation 写任何消息，也不做压缩、不做渲染、不做审计写入。这个边界很重要，因为它让快照构建可以被单独测试，也让后续排查问题时能区分“采样错误”和“渲染/预算错误”。

### render.go

`Renderer` 负责把结构化上下文转换成模型可读的 system reminder。

当前渲染的内容包括：

- 长期指令和 memory。
- Plan mode reminder。
- 通知消息。
- Active Skills SOP。
- Deferred Tools 的 ToolSearch 加载说明。

`Renderer` 的职责是“表达策略”，不是“存储策略”。它不决定这些内容是否应该被压缩，也不负责 tool schema 的实际加载。比如 deferred tool 的说明只是告诉模型可以通过 ToolSearch 选择性加载 schema，真正的 schema 暴露仍由工具系统和 Gateway 边界控制。

Plan mode 是一个特殊点：`Renderer` 会根据 `permissions.Checker` 判断是否进入 Plan mode，并同步 `Checker.PlanFilePath`。这保留了原有行为，同时把 reminder 构造从 Agent 主循环中移出。

### budgeter.go

`Budgeter` 负责上下文大小控制。

它封装两个已有底层机制：

- `compact.ManageContext` / `compact.ForceCompact`：处理 conversation 历史压缩。
- `toolresult.Apply` / `toolresult.AppendRecords`：处理大工具结果替换和落盘记录。

`Budgeter.PrepareBudget` 的输出是 `BudgetResult`，其中包括：

- `APIConversation`：预算处理后的最终 conversation。
- `ToolResultRecords`：本轮被替换的工具结果记录。
- `CompactMessage`：如果发生自动压缩，返回压缩提示。
- `CompactTracking`：更新后的压缩追踪状态。
- `UsageAnchorReset`：是否需要重置 token 使用锚点。

设计上没有把 `compact` 和 `toolresult` 直接重写到 Context Layer 里，是为了降低迁移风险。Context Layer 先成为编排者，等边界稳定后，再决定底层策略是否需要替换。

### gateway.go

`ContextGateway` 是 Agent 主循环进入 Context Layer 的唯一入口。

它组合：

- `Builder`
- `Renderer`
- `Budgeter`
- `AuditWriter`

`PrepareTurn` 的顺序是有意设计的：

1. 先注入长期 memory，确保 conversation 有稳定的长期上下文。
2. 再构建 snapshot，记录当时的结构化上下文状态。
3. 再渲染短期提醒，例如 plan、notification、active skills、deferred tools。
4. 再执行预算控制，因为渲染后的提醒也会占用上下文窗口。
5. 最后生成审计记录。

Gateway 返回 `PreparedTurn`，Agent 只拿 `APIConversation` 去流式调用模型。这样 Agent 主循环不再知道 reminder 怎么拼、工具结果怎么替换、何时压缩，只保留模型调用、工具执行和事件发布这些执行职责。

### recovery.go

`RecoveryTracker` 负责恢复材料追踪。

当前它是 `compact.RecoveryState` 的 facade，能力包括：

- 记录最近读取过的文件内容。
- 记录被激活过的 skill SOP。
- 构建 compact recovery attachment。

这个模块解决的问题是：压缩发生后，模型可能丢失近期关键上下文，例如刚读过的文件或刚加载的技能。RecoveryTracker 把这些关键材料从 conversation 历史里抽出来，作为压缩后的恢复补充。

它目前委托给 `compact.RecoveryState`，但 Agent 对外只依赖 `ContextLifecycle.Recovery()`，后续可以替换底层实现。

### lifecycle.go

`LifecycleManager` 负责上下文生命周期操作。

当前包含：

- `Recovery()`：返回恢复追踪器。
- `Clear()`：重置恢复状态。
- `ForceCompact()`：手动触发压缩。

它对应用户级或系统级生命周期动作，例如 `/clear` 和 `/compact`。设计上把这些动作从 Agent 主循环拆出来，是为了让“每轮准备”和“显式生命周期操作”有同一个上下文归属，但不混在一个函数里。

### router.go

`Router` 负责跨 agent 的上下文交接。

它当前支持两类能力：

- `BuildHandoff`：生成结构化 `HandoffPackage`。
- `BuildForkedConversation`：为 fork agent 构造 conversation。

`HandoffMode` 包括：

- `none`：不传历史消息，只记录一次交接。
- `recent`：只传最近 N 条消息。
- `summary`：传摘要，并保留当前实现中的完整消息兼容路径。
- `full`：传完整消息。
- `fork`：用于 fork agent 的 prompt-cache-sensitive replay。

Fork conversation 是最敏感的路径。它必须保留 thinking blocks、已完成的 tool use / tool result 配对，并为未完成的 tool use 插入占位 tool result，避免生成孤立工具调用。这部分从 `internal/agents` 迁移到 Router 后，交接逻辑有了统一归属。

### audit.go

`AuditWriter` 负责写上下文事件审计。

审计文件位置：

```text
.mewcode/context/audit.jsonl
```

当前事件包括：

- `context.prepare`
- `context.render`
- `context.tool_result_budget`
- `context.compact`
- `context.handoff`
- `context.tool_exposure`
- `context.recovery_file_read`
- `context.recovery_skill`

审计写入是 append-only JSONL，并且调用点必须把失败视为非致命。原因是审计不能影响 agent 的主要执行路径：如果磁盘权限、目录创建或 JSONL 写入失败，最多丢失可观测性，不应该中断模型调用或工具执行。

## 模块边界

Context Layer 的边界可以概括为：

- `conversation` 仍然是消息容器。
- `compact` 仍然是历史压缩策略。
- `toolresult` 仍然是大工具结果替换策略。
- `agent` 仍然负责模型调用、工具执行和事件发布。
- `contextmgr` 负责把上述能力组织成“准备上下文”的统一架构。

这种设计避免了一次性重写所有上下文相关逻辑。第一阶段先把入口收敛到 Gateway，把跨 agent 交接收敛到 Router，把恢复和生命周期收敛到 Lifecycle。后续如果要替换压缩算法、做更细粒度工具暴露、持久化 snapshot，改动可以集中在 Context Layer 内部。

## 后续演进方向

后续可以沿着三个方向继续增强：

- Snapshot 持久化：把 `RuntimeContext` 写入 `.mewcode/context/snapshots`，用于复盘和 resume。
- Context Policy：在 Gateway 内加入显式策略，决定哪些层级在不同 agent type 下可见。
- Handoff Filtering：让 Router 根据 mode、权限和任务类型过滤消息、工具和恢复材料，而不是只做结构化包装。

当前阶段的重点是先建立架构归属：上下文不再是 prompt 拼接的副作用，而是一个可构建、可预算、可交接、可恢复、可审计的运行时对象。
