# clawproxy
openclaw的外挂服务

## WebSocket 鉴权

`/ws` 请求现在需要：

- query 参数 `deviceId`: 设备会话 ID
- header `Authorization`: JWT token（HS256）

获取 token 的方式：

```bash
clawproxy --jwt-secret your-secret token --device-id device-1
clawproxy --jwt-secret your-secret token --device-id device-1 --expires-in 1d
```

连接示例：

```text
ws://localhost:8080/ws?deviceId=device-1
Authorization: <JWT_TOKEN>
```

当 token 缺失或校验失败时，服务会返回 `401`，并在响应中带错误码：

- `TOKEN_REQUIRED`
- `INVALID_TOKEN`

`--expires-in` 现在按“天”计算，例如 `1d`、`7d`。不传该参数时，生成的 token 默认永久有效。

## 打包脚本
新增了一个可交互/可参数化的打包脚本：

```bash
./scripts/build.sh
```

不传参数会让你选择 `mac` / `linux` / `windows`。

也可以直接指定：

```bash
./scripts/build.sh mac
./scripts/build.sh linux amd64
./scripts/build.sh windows amd64
```

默认架构：
- mac: `arm64`（适合 Mac M4）
- linux: `amd64`
- windows: `amd64`

产物输出目录：`dist/`


> 脚本会自动定位项目根目录，因此在 `scripts/` 目录里执行也可以正常打包。
