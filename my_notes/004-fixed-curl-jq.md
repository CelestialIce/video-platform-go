啊，这个问题太经典了！恭喜你，你的代码完全没有问题，你已经成功地生成了合法的预签名 URL。

你遇到的 AccessDenied 错误，和代码无关，而是和你在终端（Shell）里使用 curl 命令的方式有关。这是一个 100% 的“环境问题”或者说是“操作问题”，非常常见，我们来解决它。

问题的核心原理

你从第一个 curl 命令得到的 JSON 响应中，upload_url 的值是这样的：
"http://...?X-Amz-Algorithm=...**\u0026**X-Amz-Credential=...**\u0026**..."

问题就出在这些 \u0026 上。这是 & 符号在 JSON 字符串里的标准转义表示。

当你把这个包含了 \u0026 的字符串直接复制粘贴到下一个 curl 命令时，Shell（命令行解释器）并不会自动把它理解成 &。更糟糕的是，如果你手动把它改成了 &，Shell 会把 & 当成一个特殊字符，意思是“在后台运行命令”，这会把你的 URL 从中间切断！

最终，curl 发送给 MinIO 的 URL 是一个残缺不全的、被破坏过的 URL。MinIO 拿到这个残缺的 URL 后，去计算它的签名，发现和 URL 参数里自带的 X-Amz-Signature 完全对不上，于是出于安全考虑，它拒绝了这个请求，返回了 AccessDenied。

解决方案

我们有两种方法可以解决这个问题，推荐第一种，因为它更可靠。

方法一：将 URL 存入变量（推荐）

这种方法可以完美地避免 Shell 对特殊字符的任何错误解析。

安装 jq 工具（如果你的系统没有的话）
jq 是一个超好用的命令行 JSON 处理工具。

Generated bash
sudo apt-get install jq


分两步执行命令

第一步：登录并把 URL 存到 Shell 变量里
我们用 jq -r '.upload_url' 来从 JSON 响应中提取出 upload_url 的值，并去掉引号。
（请先确保你已经用 test@example.com 注册了）

Generated bash
# 先登录获取 Token
TOKEN=$(curl -s -X POST http://localhost:8000/api/v1/users/login \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "password": "password123"}' | jq -r '.token')

# 检查一下 Token 是否获取成功
echo "获取到的Token是: $TOKEN"

# 用获取到的Token去请求上传地址，并存入变量 UPLOAD_URL
UPLOAD_URL=$(curl -s -X POST http://localhost:8000/api/v1/videos/upload/initiate \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"file_name": "my-cool-video.mp4"}' | jq -r '.upload_url')

# 检查一下 URL 是否获取成功
echo "获取到的上传URL是: $UPLOAD_URL"
IGNORE_WHEN_COPYING_START
content_copy
download
Use code with caution.
Bash
IGNORE_WHEN_COPYING_END

第二步：使用变量进行上传
现在，UPLOAD_URL 变量里包含了干净、完整的 URL。我们可以安全地使用它。
（请确保你的测试视频路径 ~/Downloads/test.mp4 是正确的）

Generated bash
curl -T ~/Downloads/test.mp4 "$UPLOAD_URL"
IGNORE_WHEN_COPYING_START
content_copy
download
Use code with caution.
Bash
IGNORE_WHEN_COPYING_END

这一次，命令执行后应该没有任何输出，这代表上传成功！

方法二：手动修复 URL（快速但不推荐）

如果你不想安装 jq，可以手动操作，但一定要小心。

运行获取上传 URL 的命令：

Generated bash
curl -X POST http://localhost:8000/api/v1/videos/upload/initiate # (带上你的Header和Body)
IGNORE_WHEN_COPYING_START
content_copy
download
Use code with caution.
Bash
IGNORE_WHEN_COPYING_END

复制返回的 JSON 中 upload_url 的整个值（不包括两边的双引号）。

粘贴到一个文本编辑器里。

手动把里面所有的 \u0026 替换为 &。

最后，构建你的 curl 上传命令，务必、务必、务必要把整个修复好的 URL 用双引号 " 包起来！

Generated bash
# 示例，这里的 URL 是你手动修复好的
curl -T ~/Downloads/test.mp4 "http://127.0.0.1:9000/videos/raw/1/my-cool-video.mp4?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=minioadmin%2F20250623%2Fus-east-1%2Fs3%2Faws4_request&..."
IGNORE_WHEN_COPYING_START
content_copy
download
Use code with caution.
Bash
IGNORE_WHEN_COPYING_END

请尝试方法一，它更加稳妥。等你成功上传文件后（可以去 MinIO 控制台 http://localhost:9001 刷新看看 videos 桶里是否出现了 raw/1/my-cool-video.mp4），我们就可以继续流水线的最后一步：编写转码 Worker！