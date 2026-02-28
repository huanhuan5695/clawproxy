# clawproxy
openclaw的外挂服务

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
