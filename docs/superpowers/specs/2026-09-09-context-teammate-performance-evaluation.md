# Context 与 Teammate 性能及真实效果评测方案

## 1. 代码性能测试

### 1.1 测试目标

代码性能测试用于回答两个问题：

- context 管理模块在不同上下文规模下，是否能稳定、低延迟地完成消息分层、预算控制、压缩恢复和最终 API 输入构造。
- teammate 模块在多 Agent 文件邮箱、共享任务板、父子委派、对等协作场景下，是否能稳定处理并发读写、锁竞争、消息积压和任务状态流转。

这一层只衡量工程实现的性能，不直接证明 Agent 效果好坏。真实任务效果需要用第 2 章的指标单独评测。

### 1.2 Context 管理模块性能指标

- `PrepareTurn` 端到端延迟：记录 p50、p95、p99，覆盖无压缩、有压缩、工具结果很大、活跃 skill 很多等场景。
- `Builder` 构造成本：统计从 runtime state 构造 `PreparedTurn` 的耗时、分配次数、分配字节数。
- `Renderer` 渲染成本：统计 memory、instructions、plan reminder、notifications、active skills、deferred tools 渲染耗时。
- `Budgeter` 预算控制成本：统计上下文窗口接近上限时的裁剪、压缩触发、tool result 包装处理耗时。
- tool result 大对象处理成本：覆盖小结果、1MB、10MB 等输入，统计内存峰值和分配次数。
- API conversation 体积：统计最终发送给 LLM 的 message 数、总字符数、估算 token 数。
- 压缩触发频率：统计每 N 轮会话触发 compaction 的次数，避免过早压缩或漏压缩。
- 恢复路径成本：统计 skills load recovery、compaction recovery、context audit 写入耗时。

### 1.3 Teammate 模块性能指标

- `FileMailBox.Send` 写入延迟：统计单发、批量发送、并发发送下的 p50、p95、p99。
- 邮箱吞吐量：统计每秒可写入消息数、每秒可读取未读消息数。
- `ReadUnread` 与 `MarkAllRead` 成本：覆盖 inbox 为空、100 条、1000 条、10000 条消息。
- 并发发送正确性：N 个 sender 同时写入时，不允许丢消息、重复 ID、JSON 损坏。
- `TaskBoard.ClaimTask` 锁竞争：多个 Agent 抢同一个任务时，必须只有一个 winner。
- `TaskBoard.UpdateTask` 大历史成本：任务 history 很长时，更新延迟不能线性恶化到不可接受。
- `TaskBoard.ListTasks` 扫描成本：覆盖 10、1000、10000 个任务。
- lead mailbox drain 成本：统计 lead 从多个 teammate 收取消息时的延迟和消息丢失率。
- 空闲轮询 CPU：没有新消息时，teammate runner 的 CPU 占用必须接近空闲状态。
- 文件锁异常恢复：模拟 stale lock、半写入 tmp 文件、坏 JSON，验证不会导致整个 team 卡死。

### 1.4 Benchmark 数据规模

Context 管理模块：

```text
conversation messages: 10 / 100 / 1000 / 5000
tool result size: small / 1MB / 10MB
active skills: 0 / 5 / 20 / 100
deferred tools: 0 / 100 / 1000
notification count: 0 / 10 / 100
memory blocks: 0 / 10 / 100
```

Teammate 模块：

```text
teams: 1 / 10 / 100
members per team: 2 / 10 / 50
mailbox messages: 10 / 1000 / 10000
concurrent senders: 1 / 10 / 100
board tasks: 10 / 1000 / 10000
claim contention: 2 / 10 / 100 contenders
task history entries: 0 / 100 / 1000
```

### 1.5 建议新增 Benchmark

Context 管理模块：

- `BenchmarkBuilderBuild`
- `BenchmarkRendererActiveSkills`
- `BenchmarkGatewayPrepareTurn`
- `BenchmarkBudgeterLargeToolResult`
- `BenchmarkBudgeterCompactionBoundary`
- `BenchmarkGatewayPrepareTurnWithLargeConversation`
- `BenchmarkGatewayPrepareTurnWithManyDeferredTools`

Teammate 模块：

- `BenchmarkFileMailBoxSend`
- `BenchmarkFileMailBoxSendParallel`
- `BenchmarkFileMailBoxReadUnread`
- `BenchmarkFileMailBoxMarkAllRead`
- `BenchmarkTaskBoardClaimContention`
- `BenchmarkTaskBoardList`
- `BenchmarkTaskBoardUpdateLargeHistory`
- `BenchmarkDrainLeadMailbox`

### 1.6 Go 测试命令

基础 benchmark：

```bash
go test ./internal/contextmgr -bench=. -benchmem -count=10
go test ./internal/teams ./internal/orchestration -bench=. -benchmem -count=10
```

CPU 与内存 profile：

```bash
go test ./internal/contextmgr -bench=PrepareTurn -benchmem -cpuprofile cpu.out -memprofile mem.out
go tool pprof -http=:0 cpu.out
go tool pprof -http=:0 mem.out
```

文件锁、阻塞、互斥竞争：

```bash
go test ./internal/teams ./internal/orchestration -bench=FileMailBox -blockprofile block.out -mutexprofile mutex.out
go tool pprof -http=:0 block.out
go tool pprof -http=:0 mutex.out
```

runtime trace：

```bash
go test ./internal/contextmgr -run TestName -trace trace.out
go tool trace trace.out
```

并发安全：

```bash
go test ./internal/contextmgr ./internal/teams ./internal/orchestration -race -count=1
```

### 1.7 初始性能门槛

- `ContextGateway.PrepareTurn`：1000 条普通消息、无压缩时 p95 小于 20ms。
- `Renderer`：100 个 active skills 渲染 p95 小于 10ms。
- `Budgeter`：10MB tool result 处理过程不能出现非必要多倍内存复制。
- `FileMailBox.Send`：100 个并发 sender 发送后无丢消息、无坏 JSON、无重复 ID。
- `TaskBoard.ClaimTask`：100 个并发 contender 抢同一任务时，必须恰好 1 个成功。
- `DrainLeadMailbox`：10 个 team、每个 50 条消息时 p95 小于 50ms。
- 所有并发相关测试必须通过 `-race`。
- 长时间压测后不能出现 goroutine 泄漏、临时文件堆积、stale lock 阻塞。

### 1.8 代码性能测试产物

- benchmark 原始输出：保存 `go test -bench` 输出。
- profile 文件：保存 CPU、memory、block、mutex profile。
- 趋势表：按 commit 记录关键 benchmark 的 ns/op、B/op、allocs/op。
- 回归阈值：CI 中对关键 benchmark 设置可接受漂移，例如 p95 延迟不超过基线 20%。

## 2. 真实场景效果评测

### 2.1 为什么需要真实效果评测

代码性能测试只能证明模块“快”和“稳”，不能证明模块“选对了上下文”或“协作真的提高了解题率”。真实场景效果评测需要回答：

- context 管理模块选入的上下文是否包含解决任务所需信息。
- 被选入的上下文是否噪声过多，导致模型分心或预算浪费。
- 压缩、裁剪、恢复后是否丢失关键事实、约束、用户偏好和历史决策。
- teammate 协作是否提高任务成功率，而不是只增加消息数量、token 成本和等待时间。
- 文件邮箱传输的信息是否足够准确、及时、可追踪。

### 2.2 Context 真实效果指标

#### 2.2.1 Context Recall

衡量应当进入当前轮 LLM 输入的关键上下文，有多少真的被选入。

```text
Context Recall = selected_relevant_contexts / all_required_relevant_contexts
```

适用场景：

- 用户早期约束在后续轮次仍然重要。
- 历史工具结果中有关键证据。
- memory 或 plan 中有必须保留的决策。
- 压缩后需要验证摘要是否覆盖关键事实。

标注方式：

- 为每个测试任务维护 `required_context_ids`。
- 每轮 `PrepareTurn` 输出 `selected_context_ids`。
- 用 ID 级匹配计算 recall，避免 LLM judge 引入额外噪声。

#### 2.2.2 Context Precision

衡量被选入上下文中，有多少对当前任务真的有用。

```text
Context Precision = selected_relevant_contexts / all_selected_contexts
```

适用场景：

- 对比不同预算策略是否引入大量无关消息。
- 对比不同 router 是否错误保留旧工具结果。
- 检查 active skills、deferred tools、notifications 是否污染主上下文。

#### 2.2.3 Context F1

综合衡量上下文选择的完整性与纯净度。

```text
Context F1 = 2 * Precision * Recall / (Precision + Recall)
```

使用方式：

- recall 是主指标，因为漏掉关键事实通常比多带一点噪声更致命。
- precision 是成本和抗干扰指标，防止上下文膨胀。
- F1 用于比较两个策略的整体平衡。

#### 2.2.4 Context Precision@K 与 Recall@K

当 context 管理模块输出有顺序时，需要评估 top K 的质量。

```text
Precision@K = top_k_relevant / K
Recall@K = top_k_relevant / all_required_relevant_contexts
```

使用方式：

- K 可以按 token budget 切分，例如前 2K、4K、8K token。
- 排名越靠前，越应该放 memory、当前任务、近期决策、关键工具结果。

#### 2.2.5 MRR / NDCG

当只需要找到一个关键上下文时，用 MRR。

```text
MRR = mean(1 / rank_of_first_relevant_context)
```

当多个关键上下文都有不同重要性时，用 NDCG。

```text
NDCG@K = DCG@K / IDCG@K
```

使用方式：

- MRR 适合“某条历史事实是否被优先召回”。
- NDCG 适合“高价值证据是否排在低价值噪声之前”。

#### 2.2.6 Faithfulness / Groundedness

衡量最终回答是否被选入上下文支持。

适用场景：

- context recall 很高，但模型仍然编造。
- 压缩摘要引入了不存在的事实。
- teammate 传来的消息被错误引用。

评测方式：

- 优先使用引用级检查：回答中的关键断言必须能映射到 `selected_context_ids`。
- 对无法确定的语义映射，使用 LLM-as-judge，但必须保存 judge prompt、模型版本和解释。

#### 2.2.7 Constraint Retention

衡量用户约束、系统约束、开发约束、计划 checkpoint 是否在长上下文中被保留。

```text
Constraint Retention = retained_required_constraints / all_required_constraints
```

示例约束：

- 不阅读 TUI 相关代码。
- 不回滚用户改动。
- 使用文件邮箱传输 teammate 信息。
- markdown checkpoint 必须使用 `- [ ]`。

#### 2.2.8 Context Drift Rate

衡量长任务中上下文是否偏离当前目标。

```text
Context Drift Rate = drift_turns / total_turns
```

判定方式：

- 当前轮输入中缺少 active goal。
- 当前轮输入中保留了已过期目标。
- teammate 根据过期任务说明继续执行。
- 压缩摘要把历史决策解释错。

### 2.3 Teammate 真实效果指标

#### 2.3.1 Task Success Rate

衡量多 Agent 协作后是否完成最终任务。

```text
Task Success Rate = successful_tasks / total_tasks
```

任务成功应优先使用确定性检查：

- 目标文件存在且内容符合 schema。
- 测试通过。
- task board 中所有 required milestones 完成。
- mailbox 中 lead 收到必要结果。

#### 2.3.2 Agent Goal Accuracy

衡量每个 Agent 是否完成自己被委派的子目标。

```text
Agent Goal Accuracy = agents_with_goal_completed / agents_with_assigned_goal
```

适用场景：

- 父子委派：child 是否完成 parent 分配的子任务。
- 对等协作：peer 是否完成自己承诺的部分。
- 共享任务板：claim 后是否真正产出结果，而不是只改状态。

#### 2.3.3 Tool Call Precision / Recall / F1

衡量 teammate 是否调用了正确工具。

```text
Tool Precision = correct_tool_calls / actual_tool_calls
Tool Recall = correct_tool_calls / expected_tool_calls
Tool F1 = 2 * Tool Precision * Tool Recall / (Tool Precision + Tool Recall)
```

在本项目中重点评测：

- 是否正确使用 `SendMessage`、`CheckMessages`、`TaskCreate`、`TaskClaim`、`TaskUpdate`。
- 是否把父子委派误用成对等协作。
- 是否在共享任务板任务中绕过 board，直接用普通消息导致状态不可追踪。
- tool args 是否包含正确 `team_name`、`agent_name`、`task_id`、`status`。

#### 2.3.4 Collaboration Completion Rate

衡量协作协议是否走完。

```text
Collaboration Completion Rate = completed_sessions / started_sessions
```

按模式拆分：

- 父子委派：parent 创建 session，child 返回结果，parent 消化结果。
- 对等协作：peer 收到消息，回复并完成约定动作。
- 共享任务板：task 从 open 到 claimed 到 done，且有 result summary。

#### 2.3.5 Message Delivery Accuracy

衡量文件邮箱是否把消息送到了正确 Agent，并被正确理解。

```text
Delivery Accuracy = correctly_delivered_messages / sent_messages
```

错误类型：

- 发错 team。
- 发错 recipient。
- 消息格式损坏。
- 任务 ID 丢失。
- 普通消息与 assignment / board_update 混淆。
- Agent 已读但未执行。

#### 2.3.6 Coordination Overhead

衡量多 Agent 带来的额外成本。

```text
Coordination Overhead Tokens = team_tokens - single_agent_tokens
Coordination Overhead Latency = team_latency - single_agent_latency
Message Overhead = collaboration_messages / completed_tasks
```

使用方式：

- 多 Agent 方案必须和 single-agent baseline 对比。
- 如果成功率没有提升，但 token 和 latency 明显升高，应视为负收益。
- 对不同协作模式分别统计，避免平均值掩盖问题。

#### 2.3.7 Deadlock / Loop Rate

衡量 Agent 之间是否出现等待、重复询问、重复转派。

```text
Deadlock Rate = deadlocked_sessions / started_sessions
Loop Rate = repeated_coordination_cycles / total_sessions
```

可检测信号：

- task 长时间 claimed 但无更新。
- 两个 Agent 互相要求对方决策。
- 同一 task 反复 open / claimed。
- mailbox 中重复出现相同请求。
- lead drain 后没有任何可执行信息。

#### 2.3.8 Marginal Gain of Collaboration

衡量协作带来的收益是否覆盖成本。

```text
Marginal Gain = team_success_rate - single_agent_success_rate
Cost Adjusted Gain = Marginal Gain / additional_cost
```

使用方式：

- 对简单任务，预期 teammate 不应显著优于 single-agent；重点看额外成本是否可控。
- 对复杂任务，预期 teammate 应提升成功率、降低单个上下文负担。
- 对高度耦合任务，重点看 task board 是否减少信息丢失和重复劳动。

## 3. 真实场景测试集设计

### 3.1 Context 测试集

每个 case 应包含：

- `case_id`
- 用户输入序列
- 工具调用结果
- memory 初始状态
- active plan
- expected final answer 或 expected action
- `required_context_ids`
- `forbidden_context_ids`
- `required_constraints`
- 每轮期望保留的关键事实

推荐场景：

- 长对话约束保持：用户早期给出限制，20 轮后仍需遵守。
- 工具结果召回：早期工具结果中包含最终答案证据。
- 压缩后恢复：触发 compaction 后检查摘要是否保留关键事实。
- 噪声干扰：插入大量无关消息，检查 precision 和最终任务成功率。
- 多主题切换：用户从 A 任务切到 B 任务，检查旧上下文是否被降权。
- 失败恢复：工具失败后是否保留错误原因和下一步修复策略。

### 3.2 Teammate 测试集

每个 case 应包含：

- `case_id`
- collaboration mode：`delegation` / `peer` / `task_board`
- agent roles
- initial mailbox state
- initial board state
- expected messages
- expected task state transitions
- expected tool calls
- final success checker
- max allowed messages
- max allowed latency
- max allowed token cost

推荐场景：

- 父子委派：lead 要求 child 阅读某个非 TUI 模块并返回总结。
- 对等协作：两个 peer 分别分析 context 与 teams，互相交换结论后 lead 汇总。
- 共享任务板：lead 创建多个 task，teammates 抢占、更新、完成。
- 冲突处理：两个 Agent 试图 claim 同一 task，只允许一个成功。
- 邮箱积压：多个 teammate 同时发送结果，lead 必须完整 drain。
- 错误恢复：child 发送格式错误或遗漏 task_id，系统应可追踪并降级处理。
- 无收益对照：简单任务用 single-agent 与 team-agent 对比，防止协作滥用。

## 4. 评测执行流程

### 4.1 Offline Golden Eval

- 构造固定 golden set。
- 对每个 case 运行 context manager 或 teammate orchestrator。
- 收集 selected context、tool calls、mailbox messages、board events、final output。
- 使用 deterministic scorer 计算 ID-based precision、recall、F1、tool call accuracy。
- 对需要语义判断的项使用 LLM judge。
- 保存每个 case 的失败原因，按 failure taxonomy 分类。

### 4.2 Online Shadow Eval

- 从真实使用日志中抽样，但不影响线上行为。
- 回放同一输入到新策略。
- 对比旧策略与新策略的 context selection、final output、token cost、latency。
- 只在 shadow 指标稳定优于 baseline 后进入灰度。

### 4.3 A/B Eval

- 对真实任务按 session 维度分流。
- 比较 baseline 与新策略的成功率、人工干预率、成本、延迟。
- 对 teammate 模块额外比较协作消息数、deadlock rate、task completion rate。
- 对低频但严重的问题进行人工复核，例如误删关键约束、错误转派、重复执行危险操作。

### 4.4 Regression Eval

- 每次修改 context 或 teammate 模块后运行小型 golden set。
- 每周或每个 release 运行完整 golden set。
- 对历史线上失败案例做 replay，确保已经修复的问题不回归。

## 5. 推荐指标看板

Context 看板：

- Context Recall
- Context Precision
- Context F1
- Precision@K / Recall@K
- Constraint Retention
- Faithfulness / Groundedness
- Context Drift Rate
- Selected Token Count
- Compaction Trigger Rate
- PrepareTurn p95 / p99

Teammate 看板：

- Task Success Rate
- Agent Goal Accuracy
- Tool Call Precision / Recall / F1
- Collaboration Completion Rate
- Message Delivery Accuracy
- TaskBoard Completion Rate
- Deadlock / Loop Rate
- Coordination Overhead Tokens
- Coordination Overhead Latency
- Cost Adjusted Gain
- Mailbox Send / Read p95

## 6. 初始上线门槛

Context 管理模块：

- golden set Context Recall 不低于 0.95。
- Context Precision 不低于 0.75。
- Constraint Retention 不低于 0.98。
- 压缩场景关键事实丢失率低于 2%。
- Faithfulness 不低于 baseline。
- token 成本不高于 baseline 120%，除非成功率有明确提升。

Teammate 模块：

- 父子委派 Task Success Rate 不低于 single-agent baseline。
- 共享任务板场景 TaskBoard Completion Rate 不低于 0.95。
- Tool Call F1 不低于 0.90。
- Message Delivery Accuracy 等于 1.00。
- Deadlock Rate 等于 0。
- 多 Agent 成本增加必须对应成功率或复杂任务覆盖率提升。

## 7. 失败分类

Context 失败：

- `missing_required_context`：漏召回关键上下文。
- `irrelevant_context_pollution`：无关上下文过多。
- `wrong_priority_order`：关键上下文排序靠后。
- `constraint_loss`：用户或系统约束丢失。
- `compression_distortion`：压缩摘要歪曲事实。
- `stale_context_retained`：过期目标仍被保留。
- `tool_result_loss`：工具结果被裁剪或摘要时丢失关键证据。

Teammate 失败：

- `wrong_collaboration_mode`：选错父子、对等或任务板模式。
- `mail_delivery_failure`：消息未送达或送错。
- `mail_semantic_loss`：消息送达但含义丢失。
- `task_claim_conflict`：任务抢占出现多个 owner。
- `task_state_stall`：任务状态长期无进展。
- `agent_goal_miss`：Agent 未完成被委派目标。
- `coordination_loop`：重复沟通但没有推进。
- `coordination_overhead_regression`：协作成本增加但成功率无收益。

## 8. 参考来源

- Go `testing` package：benchmark、`-benchmem`、profile 与测试约定。https://go.dev/pkg/testing/
- Go diagnostics：CPU、memory、block、mutex、trace 等性能诊断路径。https://go.dev/doc/diagnostics
- Go race detector：并发安全测试。https://go.dev/doc/articles/race_detector
- Go runtime trace：运行时调度、阻塞、网络与同步分析。https://go.dev/pkg/runtime/trace/
- Ragas metrics：Context Precision、Context Recall、Faithfulness、Tool Call Accuracy、Tool Call F1、Agent Goal Accuracy 等指标。https://docs.ragas.io/en/latest/concepts/metrics/available_metrics/
- Ragas Context Precision：ID-based precision 与 ranking-based context precision。https://github.com/vibrantlabsai/ragas/blob/main/docs/concepts/metrics/available_metrics/context_precision.md
- LangSmith RAG evaluation：将 RAG 评测拆成 correctness、relevance、groundedness、retrieval relevance。https://docs.langchain.com/langsmith/evaluate-rag-tutorial
- LangSmith intermediate-step evaluation：评测 retrieval 等中间步骤，而不是只评最终输出。https://docs.langchain.com/langsmith/evaluate-on-intermediate-steps
- Phoenix RAG evaluation：将 RAG 评测拆成 retrieval evaluation 与 response evaluation。https://arizeai-433a7140.mintlify.app/docs/phoenix/cookbook/evaluation/evaluate-rag
- NVIDIA Agentic Evaluation Metrics：Tool Call Accuracy、Agent Goal Accuracy、Topic Adherence、Trajectory Evaluation。https://docs.nvidia.com/nemo-platform/documentation/evaluate-models/metrics/agentic-metrics
- AgentBench：用多环境任务成功率评测 LLM-as-Agent。https://doi.org/10.48550/arXiv.2308.03688
- Mosaic multi-agent evaluation：多 Agent 系统需要同时评估 outcome、process、coordination、overhead、deadlock/loop。https://doi.org/10.1145/3772363.3798830

## 9. 执行命令索引

详细命令见 `docs/superpowers/specs/2026-09-09-context-teammate-performance-commands.md`。
