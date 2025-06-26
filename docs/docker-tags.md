# Docker 镜像标签配置说明

## 标签策略

本项目的 GitHub Actions 会自动为 Docker 镜像生成多种标签，方便不同场景下的使用。

### 基础标签

| 标签格式 | 示例 | 说明 |
|---------|------|------|
| `latest` | `username/xiaohu-jobs:latest` | 最新版本，始终指向最新的构建 |
| `{branch}` | `username/xiaohu-jobs:main` | 分支名称，如 main、develop |
| `v{build}` | `username/xiaohu-jobs:v123` | 构建编号，每次构建递增 |
| `{commit}` | `username/xiaohu-jobs:a1b2c3d` | Git 提交哈希，精确到具体提交 |
| `{date}` | `username/xiaohu-jobs:20241201-143022` | 构建时间戳 |

### 发布版本标签（仅限 Git Tags）

当推送 Git Tag 时，会额外生成语义化版本标签：

| 标签格式 | 示例 | 说明 |
|---------|------|------|
| `{version}` | `username/xiaohu-jobs:v1.2.3` | 完整版本号 |
| `{major}.{minor}` | `username/xiaohu-jobs:v1.2` | 主版本.次版本 |
| `{major}` | `username/xiaohu-jobs:v1` | 主版本 |

## 使用示例

### 拉取最新版本
```bash
docker pull username/xiaohu-jobs:latest
```

### 拉取特定分支版本
```bash
docker pull username/xiaohu-jobs:main
```

### 拉取特定构建版本
```bash
docker pull username/xiaohu-jobs:v123
```

### 拉取发布版本
```bash
docker pull username/xiaohu-jobs:v1.2.3
```

## 自定义配置

### 修改镜像名称

在 `.github/workflows/deploy.yml` 中修改：

```yaml
env:
  IMAGE_NAME: 'your-custom-name'
```

### 修改标签策略

可以在 `生成 Docker 标签` 步骤中自定义标签生成逻辑：

```bash
# 添加自定义标签
CUSTOM_TAG="${IMAGE_NAME}:custom-${GITHUB_RUN_NUMBER}"
echo "custom_tag=${CUSTOM_TAG}" >> $GITHUB_OUTPUT
```

### 使用私有镜像仓库

修改 `DOCKER_REGISTRY` 环境变量：

```yaml
env:
  DOCKER_REGISTRY: 'your-registry.com'
```

## 标签优先级

1. **latest** - 生产环境推荐
2. **v{version}** - 发布版本推荐
3. **v{build}** - 测试环境推荐
4. **{commit}** - 调试特定问题
5. **{date}** - 时间回溯

## 注意事项

- 所有标签都指向同一个镜像层，只是标签不同
- `latest` 标签会覆盖之前的 `latest` 标签
- 建议在生产环境使用具体的版本标签，而不是 `latest`
- 定期清理旧的标签以节省存储空间 