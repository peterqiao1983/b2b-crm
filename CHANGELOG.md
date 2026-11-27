# Changelog

本项目的所有重要变更都会记录在此文件。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，版本遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [1.0.0] - 2026-11-27

首个正式版本，包含完整的 B2B CRM 前后端。

### 新增

- 完整 B2B CRM：客户（公司）、联系人、项目、跟进记录、日历待办
- 登录鉴权（JWT）+ 登录限流
- 仪表盘总览（公司 / 项目 / 联系人统计）
- 公司链式详情（`chain-summary`）
- 软删除与回收站恢复（`/trash`）
- 审计日志（`/audit-logs`，记录增删改操作）
- CSV 数据导出（`/export/csv`）
- 看板视图（按阶段拖拽管理项目）
- 移动端响应式布局
- stdio MCP server（供 Hermes work assistant 集成查询 / 更新）
- 独立联系人：联系人可独立存在，也可关联公司（`?unlinked=1` 过滤）
- 跟进记录弹窗「对接人」下拉内联新增对接人，自动回填选中
- 日历待办详情弹窗「取消待办」按钮，一键清空下次跟进时间

### 修复

- SQLite `PRAGMA table_info` 的 notnull 字段映射错误

### 移除

- 跟进弹窗「对接人总表」链接（用户反馈不需要）
