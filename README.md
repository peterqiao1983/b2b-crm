# B2B CRM

自用的 B2B 客户关系管理系统，前后端分离，支持手机浏览器直接使用。

## 功能特性

- **客户管理**：公司档案、软删除回收站（可恢复）
- **联系人管理**：联系人可独立存在，也可关联到公司
- **项目管理**：项目关联公司，项目与联系人多对多关联
- **跟进记录**：跟进方式、纪要、下次跟进时间排期
- **日历 / 待办**：跟进日历视图，待办按「下次跟进时间」自动排期，可一键取消
- **跟进弹窗内联新增对接人**：写跟进记录时直接新建联系人并自动回填
- **看板视图**：按阶段拖拽管理项目
- **审计日志**：记录所有增删改操作（操作人 + 类型 + 内容）
- **CSV 导出**：客户 / 联系人 / 项目数据一键导出
- **登录鉴权**：JWT 鉴权 + 登录限流
- **移动端适配**：响应式布局，手机浏览器可直接使用
- **MCP Server**：`stdio` 服务，供 Hermes work assistant 集成查询 / 更新

## 技术栈

- **前端**：Vue 3 + Element Plus + Vite + FullCalendar + Pinia + vue-router
- **后端**：Go + Gin + SQLite（WAL 模式）
- **MCP**：stdio server（`backend/cmd/mcp`）

## 数据模型

| 实体 | 说明 |
| --- | --- |
| `User` | 登录用户 |
| `Company` | 客户公司 |
| `Contact` | 联系人（可独立或关联公司） |
| `Project` | 项目（关联公司 + 阶段） |
| `ProjectContact` | 项目 — 联系人关联 |
| `FollowUp` | 跟进记录（含下次跟进时间） |
| `AuditLog` | 审计日志 |

## 目录结构

```
b2b-crm/
├── frontend/          # Vue 3 前端
│   └── src/
│       ├── views/     # Dashboard / Companies / Contacts / Projects / Calendar / Login / Layout
│       ├── components/  # CompanyChainDrawer（公司链式详情）
│       └── api/       # 后端 API 客户端
└── backend/           # Go 后端
    ├── cmd/
    │   ├── server/    # HTTP 服务入口
    │   └── mcp/       # MCP stdio server
    └── internal/
        ├── handlers/  # auth / crm 处理
        ├── middleware/ # 鉴权 / 限流
        ├── models/    # 数据模型
        └── db/        # SQLite 连接
```

## 部署

- **域名**：`https://crm.example.com`（Cloudflare 橙云 + nginx 反代）
- **前端**：静态文件 `/var/www/crm-frontend/dist`（nginx）
- **后端**：`/opt/crm/crm-server`，systemd `crm.service`，监听 127.0.0.1:8080
- **数据库**：`/opt/crm/data/crm.db`
- **MCP**：`/opt/crm/crm-mcp`

## 开发

后端编译（在家庭服务器 home-bastion）：

```bash
cd backend
go build -o crm-server ./cmd/server
```

前端编译（注意：home-bastion 无 node/npm，需在 aws-my 本机用 Hermes 内置 node/npm）：

```bash
export PATH=/home/admin/.hermes/tools/node-26.12.1-linux-x64/bin:/home/admin/.hermes/tools/npm-12.2.2-linux-x64/bin:$PATH
cd frontend
npm install
npm run build   # 产物在 dist/
```

前端改版后需将 `dist/` 部署到 `/var/www/crm-frontend/dist`，浏览器强刷（Ctrl+F5）生效。

## API 一览（前缀 `/api/v1`）

- 认证：`POST /auth/login`、`GET /auth/me`、`POST /auth/change-password`
- 仪表盘：`GET /dashboard`
- 公司：`GET/POST /companies`、`PUT/DELETE /companies/:id`、`GET /companies/:id/chain-summary`
- 联系人：`GET/POST /contacts`、`PUT/DELETE /contacts/:id`
- 项目：`GET/POST /projects`、`PUT/DELETE /projects/:id`、`POST /projects/:id/contacts`、`DELETE /projects/:id/contacts/:contactId`
- 跟进：`GET/POST /follow-ups`、`PUT/DELETE /follow-ups/:id`
- 日历：`GET /calendar/follow-ups`
- 回收站：`GET /trash`、`POST /trash/restore`
- 审计：`GET /audit-logs`
- 导出：`GET /export/csv`
