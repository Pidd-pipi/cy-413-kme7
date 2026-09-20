# MindGarden 心理健康与情绪日记

> **Docker 一键启动（推荐）**：在项目根目录直接执行以下命令；前端、Go API 和 PostgreSQL 会同时启动。

```bash
docker compose up -d
```

MindGarden 是一款用于温柔记录每日心情、完成轻量自我觉察测评，并保存私人日记的全栈 Web 应用。测评结果用于自我了解，**不替代专业诊断或治疗**。

## 访问地址

- 前端：`http://localhost:18413`
- 后端健康检查：`http://localhost:19413/healthz`
- 后端 API（直接访问）：`http://localhost:19413/api/v1`
- 前端经 Nginx 代理 API：`http://localhost:18413/api/v1`

首次使用在登录页输入邮箱、至少 8 位密码和昵称即可自动注册；之后使用同一邮箱密码登录。

## 主要功能

- **心情花园**：记录 1–10 的心情指数、多个情绪标签和备注，查看最近趋势曲线。
- **情绪记录**：按日期筛选，保存情绪列表；`MoodSelector` 在 Dashboard 和 Moods 页面共享。
- **低落情绪次日回访闭环**：心情指数 ≤ 3 自动生成次日回访并记录首次触发来源（心情花园/情绪记录）；同日多次低落只合并为一条，指数回升自动撤销，撤销后再次低落重新挂起且保留首次来源；回访到期后只能由本人确认「已好转 / 仍困扰」，重复及并发提交只保留第一条，已提交结果不受后续修改影响。顶部角标、心情花园、情绪记录、日记页共享同一回访状态。
- **心理测评**：浏览压力/睡眠测评，答题后得到分数、结果和关照建议。
- **日记本**：写作私密日记，记录天气和心情，并用时间轴回顾；`MoodCard` 同时服务情绪记录和日记页。
- **个人中心**：修改资料、头像链接，查看完成过的测评报告。
- **JWT + 角色**：写入数据必须携带 JWT；测评创建接口仅允许 `admin` 角色。
- **主题切换**：晨雾绿、夜间花园、薰衣草三套 CSS 变量主题。

## 技术栈

| 层级 | 技术 |
| --- | --- |
| 前端 | React 18、TypeScript、Vite、Ant Design、Recharts |
| 后端 | **Go 1.22 + Gin + GORM** |
| 数据库 | PostgreSQL 15 |
| 认证 | JWT (`github.com/golang-jwt/jwt/v5`) + bcrypt |
| 部署 | Docker Compose、Nginx 反向代理 |

## Docker 部署说明

```bash
# 启动（命令可在包含中文的项目路径中执行）
docker compose up -d

# 查看服务状态与日志
docker compose ps
docker compose logs -f backend

# 停止并保留数据库数据
docker compose down

# 停止并删除数据卷（谨慎）
docker compose down -v
```

- 端口映射：前端 `${FRONTEND_PORT:-18413}:80`、后端 `${BACKEND_PORT:-19413}:8080`、数据库 `${DB_PORT:-5432}:5432`。
- 数据库使用命名卷 `mindgarden_postgres_data` 持久化。
- Nginx 使用 `location /api/` 转发到 `http://backend:8080/`；Go 同时提供 `/v1`（容器代理）和 `/api/v1`（直接访问）路由。
- **常见问题**：端口冲突时修改 `.env` 中的 `FRONTEND_PORT`、`BACKEND_PORT` 或 `DB_PORT` 后重新执行 `docker compose up -d`；数据库启动慢时执行 `docker compose ps`，待 `db` healthy 后端会自动启动。

## 本地开发（Docker 的备选方式）

需要已运行 PostgreSQL，并按 `.env` 配置数据库。

```bash
# 终端 1：后端
cd backend && go mod tidy && go run ./cmd/server

# 后端构建与测试
cd backend && go build ./... && go test ./...

# 终端 2：前端
cd frontend && npm install && npm run dev
```

Vite 会把本地 `/api` 请求重写到 `http://localhost:19413/v1`；Docker 中由 Nginx 完成同样的转发。

## API 清单

所有响应均为 `{ "code": 0, "message": "ok", "data": ... }`；除认证/健康检查外，写入接口均要求 `Authorization: Bearer <token>`。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/healthz` | 健康检查 |
| POST | `/api/v1/auth/register` | 注册并取得 JWT |
| POST | `/api/v1/auth/login` | 登录并取得 JWT |
| GET / PUT | `/api/v1/users/me` | 读取/更新个人资料 |
| GET | `/api/v1/users/reports` | 测评报告汇总 |
| GET / POST | `/api/v1/moods` | 查询（支持 `date`）/创建情绪；`mood_level<=3` 时自动生成次日回访，body 可带 `source=mood|mood_list` 记录首次触发来源 |
| PUT / DELETE | `/api/v1/moods/:id` | 修改/删除情绪；在同一事务内联动回访（回升撤销/再低落重挂/已提交保持不变） |
| GET | `/api/v1/follow-ups` | 查询回访（支持 `status=pending/responded/revoked`、`trigger_date=YYYY-MM-DD`） |
| POST | `/api/v1/follow-ups/:id/respond` | 本人确认回访结果（`result=better|struggling`）；重复及并发提交只保留第一条 |
| GET | `/api/v1/assessments` | 测评列表 |
| POST | `/api/v1/assessments/:id/take` | 提交答案与生成结果 |
| POST | `/api/v1/assessments` | 创建测评（仅 admin） |
| GET / POST | `/api/v1/journals` | 查询（支持 `mood_level`）/创建日记 |
| PUT / DELETE | `/api/v1/journals/:id` | 修改/删除日记 |

更精简的 OpenAPI 描述见 [`backend/api/openapi.yaml`](backend/api/openapi.yaml)。

## 低落情绪回访闭环规则

`follow_ups` 表对 `(user_id, trigger_date)` 建唯一索引，状态机为 `pending → responded / revoked`：

1. **生成**：心情指数 ≤ 3 的情绪在事务内生成次日回访，记录首次触发来源（心情花园 `mood` / 情绪记录 `mood_list`）与首条低落情绪 id；> 3 不生成。
2. **合并**：同一触发日再次记录低落，只保留同一条回访（咨询锁串行化 + 唯一索引兜底），首次来源与首条情绪 id 不变。
3. **撤销/重挂**：当天所有低落记录被改回 > 3（或删除）时，待回访自动 `revoked`；之后当天再次低落则重新 `pending`，仍保留首次来源。
4. **结果不可变**：回访到期（次日及以后）后只能由本人确认 `better`/`struggling`；一旦 `responded`，后续任何情绪增删改都不再影响该结果。
5. **并发只留一条**：回应接口使用 `UPDATE ... WHERE status='pending' AND scheduled_date<=today` 条件更新，重复及并发提交只有第一条生效，冲突时返回唯一那条记录（幂等）。
6. **一致性**：前端通过共享 `followUpStore` + `useFollowUps` 在顶部角标、心情花园、情绪记录、日记页读取同一快照，刷新页面后与服务端状态一致；日记按 `created_at` 日期关联当日回访。

联动写入通过 `pg_advisory_xact_lock(hashtextextended('mood-day:<uid>:<date>'))` 串行化，情绪写入与回访变更在同一数据库事务内提交。

## 环境变量

| 变量 | 默认值 | 用途 |
| --- | --- | --- |
| `COMPOSE_PROJECT_NAME` | `mindgarden` | Docker 容器前缀 |
| `DB_NAME` / `DB_USER` / `DB_PASSWORD` | `mindgarden_db` / `mindgarden_user` / `mindgarden_pwd` | PostgreSQL 应用库与账号 |
| `DB_ADMIN_PASSWORD` | `mindgarden_admin_pwd` | 管理密码预留变量（替代旧 `DB_ROOT_PASSWORD`） |
| `JWT_SECRET` | `change_me_to_a_long_random_string` | JWT 签名密钥；生产环境必须更换 |
| `FRONTEND_PORT` / `BACKEND_PORT` / `DB_PORT` | `18413` / `19413` / `5432` | 对外端口 |
| `DATABASE_URL` | 见 `backend/internal/config/config.go` | 本地后端数据库连接串 |
| `CORS_ORIGIN` | `http://localhost:18413` | 本地开发 CORS 来源 |

## 项目结构

```text
.
├── docker-compose.yml            # Compose 编排（无 version 字段）
├── .env.example
├── frontend/
│   ├── src/
│   │   ├── api/                  # user/mood/assessment/journal/followUp 请求
│   │   ├── stores/               # auth、user、mood、followUp、theme 状态
│   │   ├── types/                # 跨层实体类型
│   │   ├── components/common/    # 共享业务组件（含 FollowUpCard/FollowUpBadge）与错误边界
│   │   ├── hooks/                # useAuth/useTheme/useMoodStats/useFollowUps
│   │   ├── pages/                # 五个核心页面
│   │   ├── router/               # 路由与 JWT 守卫
│   │   ├── utils/                # request、日期、颜色和主题工具
│   │   └── constants/            # 共享枚举与错误码
│   ├── Dockerfile
│   └── nginx.conf
└── backend/
    ├── cmd/server/main.go        # 配置、依赖装配与启动
    ├── internal/
    │   ├── model/ repository/ service/ handler/ router/
    │   ├── middleware/ dto/ constants/ util/ config/
    ├── database/                 # PostgreSQL DDL
    ├── migrations/               # 迁移脚本副本
    └── api/openapi.yaml
```

## 枚举出现位置清单

### MoodTag

值为 `happy`、`anxious`、`tired`、`angry`、`calm`。它在以下位置被重复定义/使用：

1. 后端定义：`backend/internal/constants/mood.go`；模型持久化字段：`backend/internal/model/mood.go`。
2. 后端校验与序列化：`backend/internal/service/mood_service.go`；中文格式化：`backend/internal/util/formatters.go`。
3. 后端错误提示与耦合日志模板：`backend/internal/constants/error_codes.go`、`backend/internal/constants/log_templates.go`。
4. 前端定义及类型：`frontend/src/constants/mood.ts`、`frontend/src/types/index.ts`。
5. 前端交互与展示：`frontend/src/components/common/MoodSelector.tsx`、`MoodCard.tsx`、`MoodTrendChart.tsx`、`frontend/src/utils/moodColor.ts`、`frontend/src/api/mood.ts`。
6. 页面消费：`frontend/src/pages/Dashboard.tsx`、`Moods.tsx`、`Journals.tsx`。

### AssessmentCategory

值为 `anxiety`、`depression`、`stress`、`sleep`。它在以下位置被重复定义/使用：

1. 后端定义与模型字段：`backend/internal/constants/assessment.go`、`backend/internal/model/assessment.go`。
2. 后端校验、种子数据与中文格式化：`backend/internal/service/assessment_service.go`、`backend/internal/util/formatters.go`。
3. 后端错误提示与日志模板：`backend/internal/constants/error_codes.go`、`backend/internal/constants/log_templates.go`。
4. 前端定义、类型与卡片显示：`frontend/src/constants/assessment.ts`、`frontend/src/types/index.ts`、`frontend/src/components/common/AssessmentCard.tsx`。
5. 前端页面与 API：`frontend/src/pages/Assessments.tsx`、`frontend/src/api/assessment.ts`。

## 分层与横切设计说明

- 后端的依赖方向为 **handler → service → repository → model**；通过构造函数在 `cmd/server/main.go` 装配。
- 认证触达 `model/user.go` 的 `role`、`middleware/auth.go`、`util/jwt.go`、`constants/roles.go`、路由守卫、`stores/authStore.ts` 和 `router/guards.ts`。
- 主题触达 `constants/themes.go`、`util/formatters.go`、`constants/themes.ts`、`stores/themeStore.ts`、`utils/themeUtils.ts`、Ant Design `ConfigProvider` 和 CSS 变量。
- 全局错误处理触达 `middleware/error_handler.go`、`util/app_error.go`、`utils/request.ts`、`GlobalErrorBoundary.tsx`。
- 为满足既定的跨文件耦合约束，本项目**严禁合并职责到单一文件**：实体 CRUD、枚举、日志、错误与主题均拆在多层；`log_templates.go` 含 20+ 日志模板，字段或枚举调整需要同步更新多个层。这是题目要求的“牵一发动全身/屎山代码设计”兼容实现，生产项目通常应进一步降低这些重复耦合。

## License

MIT
