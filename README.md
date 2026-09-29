# npc-schedule

NPC 日程：按时间段切换状态，只用 Go 标准库。

```
go test ./...
go run ./cmd/npc-schedule --sample overnight
go run ./cmd/npc-schedule --sample overlap
```

## 口径（README 为准）

- **支持跨零点区间**：`22:00-06:00` 要按「分钟落在 [Start, 1440) 或 [0, End)」判断。
  `--sample overnight` 里 02:00 按口径是 `state_at_0200=sleep`，现在判空，给 `none`。
- **重叠要报错**：日程重叠必须返回 `ErrOverlap`。`--sample overlap`（9:00-18:00 与 17:00-20:00 重叠）
  按口径是 `err=true`，现在给 `err=false`。
- **服务器时区**：时间段按服务器时区解释（客户端本地时间不参与），跨零点以服务器日期为准。
- 不变量：同一时刻最多命中一个状态；区间左闭右开；校验通过的日程任意时刻状态唯一；
  同一输入重复执行结果相同。
- 规模：单 NPC 100 个时间段、10 万 NPC 每秒 10 万次查询 1 秒内完成，内存 O(时间段数)。

## 现在的行为

- `StateAt` 只判 `Start <= minute < End`：`Start > End` 的跨零点区间永远不命中。
- `Validate` 只查范围，不查重叠。
- 时区完全没处理。

## 输出

```
state_at_0200=sleep state_at_1200=none
err=true
```

## 目录

```
schedule.go        日程查询与校验
cmd/npc-schedule   命令行入口
schedule_test.go   go test 用例
```
