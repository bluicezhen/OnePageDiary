# OnePage Diary

离线优先的日记 PWA + 自建同步服务（Go + SQLite）。

## 结构

- `web/` 前端（Vue 3 + Vite + TS）
- `server/` 后端（Go + GORM + SQLite）

## 本地运行

### 后端

```bash
cd server
# go1.24
go mod tidy
go run ./cmd/server
```

默认地址 `http://127.0.0.1:8080`，默认账号 `admin/admin`。

环境变量：

- `APP_ADDR` 监听地址，默认 `:8080`
- `APP_DB_PATH` SQLite 路径，默认 `./data/diary.db`
- `APP_ADMIN_USER` / `APP_ADMIN_PASS`
- `APP_JWT_SECRET` JWT 签名密钥
- `APP_CORS_ORIGINS` 允许跨域，默认 `*`

### 前端

```bash
cd web
# node 24 LTS
npm install
npm run dev
```

浏览器打开 `http://127.0.0.1:5173`。
开发模式下 `/api` 会代理到 `http://127.0.0.1:8080`。

### Docker Compose

```bash
docker compose up --build
```

前端地址：`http://127.0.0.1:8081`（通过 `/api` 反代后端，默认无需手动填写服务端地址）。

## 同步说明

- Revision：服务端自增 `uint64`
- Cursor：SyncEvents 自增 ID
- 冲突：保留本地与远端两份，需手动选择覆盖策略

## 注意

- 当前仅实现 HTTP，HTTPS 请用反向代理。
- JWT 默认不过期，请自行设置安全策略。
