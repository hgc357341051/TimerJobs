# GitHub Actions 构建功能说明

## 🚀 当前构建功能

### 1. 基础构建任务
- **构建和测试**: 编译代码、运行单元测试、生成覆盖率报告
- **代码质量检查**: 使用 golangci-lint 进行代码质量分析
- **多平台构建**: 支持多个操作系统和架构

### 2. 多平台支持
| 平台 | 架构 | 文件扩展名 | 说明 |
|------|------|------------|------|
| Linux | amd64 | 无 | 主流服务器平台 |
| Linux | arm64 | 无 | ARM 服务器平台 |
| Windows | amd64 | .exe | Windows 桌面/服务器 |
| macOS | amd64 | 无 | Intel Mac |
| macOS | arm64 | 无 | Apple Silicon Mac |

### 3. 安全扫描
- **依赖漏洞扫描**: 使用 `govulncheck` 检查依赖包安全漏洞
- **代码安全分析**: 使用 `gosec` 进行代码安全分析
- **容器镜像安全**: 容器镜像安全扫描（可扩展）

### 4. 性能测试
- **基准测试**: 运行 Go 基准测试，检测性能回归
- **内存泄漏检测**: 使用 `-race` 标志检测竞态条件
- **内存使用分析**: 分析内存分配和使用情况

### 5. 文档生成
- **API 文档**: 自动生成 Swagger API 文档
- **代码文档**: 生成代码文档和注释
- **构建产物**: 上传文档到 GitHub Actions 产物

## 🔧 构建流程

```
1. 检出代码
2. 设置 Go 环境
3. 安装依赖
4. 运行测试
5. 代码质量检查
6. 多平台构建
7. 安全扫描
8. 性能测试
9. 文档生成
10. 通知结果
```

## 📊 构建产物

### 可执行文件
- `xiaohu-jobs-linux-amd64` - Linux x86_64
- `xiaohu-jobs-linux-arm64` - Linux ARM64
- `xiaohu-jobs-windows-amd64.exe` - Windows x86_64
- `xiaohu-jobs-darwin-amd64` - macOS Intel
- `xiaohu-jobs-darwin-arm64` - macOS Apple Silicon

### 文档
- `coverage.html` - 测试覆盖率报告
- `docs/` - API 文档和代码文档

## ⚙️ 配置选项

### 环境变量
```yaml
env:
  GO_VERSION: '1.24'          # Go 版本
  PROJECT_NAME: 'xiaohu-jobs' # 项目名称
```

### 触发条件
- **推送**: 推送到 main/master 分支
- **拉取请求**: 创建 PR 到 main/master 分支
- **手动触发**: 通过 workflow_dispatch 手动触发

## 🎯 使用建议

### 1. 日常开发
- 推送代码到功能分支
- 创建 PR 到主分支
- 自动运行所有检查

### 2. 发布版本
- 创建 Git Tag
- 自动构建发布版本
- 生成多平台可执行文件

### 3. 性能监控
- 定期运行基准测试
- 监控性能回归
- 分析内存使用

### 4. 安全维护
- 定期更新依赖
- 运行安全扫描
- 修复发现的问题

## 🔍 故障排除

### 常见问题
1. **构建失败**: 检查代码语法和依赖
2. **测试失败**: 查看测试日志，修复失败的测试
3. **质量检查失败**: 根据 linter 建议修复代码
4. **安全扫描警告**: 评估安全风险，必要时修复

### 调试方法
1. 查看 GitHub Actions 日志
2. 本地运行相同的命令
3. 检查环境变量和配置
4. 联系维护者获取帮助

## 📈 扩展功能

### 可添加的功能
- **Docker 多架构构建**: 构建多架构 Docker 镜像
- **自动部署**: 自动部署到测试/生产环境
- **通知集成**: 集成 Slack、钉钉等通知
- **性能基准**: 建立性能基准线
- **依赖更新**: 自动检查和更新依赖

### 集成建议
- **SonarQube**: 代码质量分析
- **Trivy**: 容器安全扫描
- **Prometheus**: 性能监控
- **Grafana**: 可视化监控

---

## 🚀 自动发布功能

### GitHub Releases 自动发布
当您推送 Git 标签时，系统会自动：
1. **构建多平台二进制文件** - Linux、Windows、macOS
2. **生成 API 文档** - 自动生成 Swagger 文档
3. **创建发布包** - 打包为 `.tar.gz` 和 `.zip`
4. **发布到 GitHub Releases** - 自动创建 Release 页面

### GitHub Packages 自动发布
同时还会：
1. **构建 Docker 镜像** - 多架构支持
2. **发布到 GitHub Container Registry** - 自动推送镜像
3. **生成版本标签** - 语义化版本标签

### 使用方法
```bash
# 创建版本标签
git tag v1.0.0
git push origin v1.0.0
```

### 详细说明
请查看 [自动发布功能说明](releases.md) 了解完整的使用方法和配置选项。 