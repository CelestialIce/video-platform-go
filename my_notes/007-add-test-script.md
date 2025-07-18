好的，没问题！为你整理一份完整、可复制粘贴的端到端测试流程脚本，并解释每一步的作用。你可以将这些命令保存下来，随时用于回归测试。

我们将使用 `bash` 脚本，并借助 `jq` 工具来自动提取和传递变量（如 Token, Video ID），这样就无需手动复制粘贴，非常高效。

---

### **完整测试流程脚本 (`full_test.sh`)**

你可以创建一个名为 `full_test.sh` 的文件，将下面的所有内容粘贴进去，然后通过 `bash full_test.sh` 来一键执行。

```bash
#!/bin/bash

# ==============================================================================
#           视频平台后端 MVP - 端到端完整功能测试脚本
# ==============================================================================
#
# 使用方法:
# 1. 确保你的 API Server 和 Worker 正在运行。
# 2. 确保你的环境中安装了 `curl` 和 `jq`。
# 3. 创建一个用于上传的测试视频文件，并修改下面的 `TEST_VIDEO_PATH` 变量。
# 4. 在终端中运行 `./full_test.sh` (或者 `bash full_test.sh`)
#
# ==============================================================================

# --- 变量定义 ---
API_BASE_URL="http://localhost:8000/api/v1"
TEST_VIDEO_PATH="~/Downloads/test.mp4" # <--- !!! 修改成你自己的测试视频路径 !!!
TEST_VIDEO_FILENAME=$(basename "$TEST_VIDEO_PATH")

# 用于存储会话信息的文件
SESSION_FILE="login_response.json"

# --- 辅助函数 ---
function print_header() {
    echo ""
    echo "=================================================="
    echo "  $1"
    echo "=================================================="
}

function print_command() {
    echo "CMD> $1"
}

function print_info() {
    echo "INFO> $1"
}


# ==============================================================================
#                              测试开始
# ==============================================================================

# --- 阶段 1: 用户认证 ---

print_header "1. 用户注册与登录"

# 1.1 注册一个新用户 (如果已存在会失败，但不影响后续登录)
USER_EMAIL="testuser-$(date +%s)@example.com"
print_command "注册新用户: $USER_EMAIL"
curl -s -X POST "$API_BASE_URL/users/register" \
    -H "Content-Type: application/json" \
    -d "{
        \"nickname\": \"PowerUser\",
        \"email\": \"$USER_EMAIL\",
        \"password\": \"password123\"
    }"
echo ""

# 1.2 使用一个固定账户登录来获取 Token
print_command "登录固定账户: test@example.com"
LOGIN_RESPONSE=$(curl -s -X POST "$API_BASE_URL/users/login" \
    -H "Content-Type: application/json" \
    -d '{
        "email": "test@example.com",
        "password": "password123"
    }')

# 检查登录是否成功
if ! echo "$LOGIN_RESPONSE" | jq -e '.token' > /dev/null; then
    print_info "登录失败! 请确保 'test@example.com' 用户已注册，密码为 'password123'。"
    exit 1
fi

# 将登录信息保存到文件，并提取 Token
echo "$LOGIN_RESPONSE" > $SESSION_FILE
TOKEN=$(jq -r '.token' $SESSION_FILE)
print_info "登录成功，Token 已提取并存入 $SESSION_FILE"


# --- 阶段 2: 视频处理流水线 ---

print_header "2. 视频上传与转码流水线"

# 2.1 申请预签名上传URL
print_command "申请上传URL..."
INITIATE_RESPONSE=$(curl -s -X POST "$API_BASE_URL/videos/upload/initiate" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"file_name\": \"$TEST_VIDEO_FILENAME\"}")

UPLOAD_URL=$(echo "$INITIATE_RESPONSE" | jq -r '.upload_url')
VIDEO_ID=$(echo "$INITIATE_RESPONSE" | jq -r '.video_id')

if [ "$UPLOAD_URL" == "null" ] || [ "$VIDEO_ID" == "null" ]; then
    print_info "申请上传URL失败! 响应: $INITIATE_RESPONSE"
    exit 1
fi
print_info "申请成功! Video ID: $VIDEO_ID"
print_info "Upload URL: ${UPLOAD_URL:0:100}..." # 只显示部分URL

# 2.2 使用预签名URL上传文件
print_command "使用 cURL 上传本地视频文件..."
# 必须将 ~ 符号展开为实际的家目录路径
EXPANDED_VIDEO_PATH=$(eval echo "$TEST_VIDEO_PATH")
UPLOAD_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -T "$EXPANDED_VIDEO_PATH" "$UPLOAD_URL")

if [ "$UPLOAD_STATUS" -ne 200 ]; then
    print_info "文件上传到 MinIO 失败! HTTP 状态码: $UPLOAD_STATUS"
    exit 1
fi
print_info "文件上传成功! (HTTP 200)"

# 2.3 通知服务器上传完成，触发转码
print_command "通知服务器上传完成，触发后台转码..."
curl -s -X POST "$API_BASE_URL/videos/upload/complete" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"video_id\": $VIDEO_ID}"
echo "" # 换行
print_info "转码任务已提交! 请在 Worker 终端观察日志。"
print_info "等待 20 秒，以便 Worker 完成转码..."
sleep 20


# --- 阶段 3: 结果验证与内容查询 ---

print_header "3. 验证转码结果与查询 API"

# 3.1 查询视频详情，验证转码成果
print_command "查询 Video ID: $VIDEO_ID 的详情..."
VIDEO_DETAILS=$(curl -s -X GET "$API_BASE_URL/videos/$VIDEO_ID")
echo "$VIDEO_DETAILS" | jq # 使用 jq 美化输出

print_info "--- 请检查以上 JSON 输出 ---"
print_info "1. 'status' 应该为 'online'。"
print_info "2. 'duration' 应该有一个大于0的数值。"
print_info "3. 'cover_url' 应该包含 'cover.jpg'。"
print_info "4. 'sources' 数组应该包含 '360p' 和 '720p' 两个对象。"
print_info "5. 'sources' 中的 'url' 应该是带签名的临时播放地址。"


# 3.2 测试评论功能
print_command "为 Video ID: $VIDEO_ID 发表一条评论..."
curl -s -X POST "$API_BASE_URL/videos/$VIDEO_ID/comments" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"content": "这是通过自动化脚本发表的评论！"}' | jq
echo ""

print_command "再次为 Video ID: $VIDEO_ID 发表一条弹幕..."
curl -s -X POST "$API_BASE_URL/videos/$VIDEO_ID/comments" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"content": "弹幕来啦~~~", "timeline": 15}' | jq
echo ""

print_command "查询 Video ID: $VIDEO_ID 的评论列表..."
COMMENTS_LIST=$(curl -s -X GET "$API_BASE_URL/videos/$VIDEO_ID/comments")
echo "$COMMENTS_LIST" | jq

print_info "--- 请检查以上 JSON 输出 ---"
print_info "1. 应该包含两条评论记录。"
print_info "2. 每条评论都应该有一个 'user' 对象，内含 'id' 和 'nickname'。"


# 3.3 测试视频列表
print_command "查询视频列表..."
curl -s -X GET "$API_BASE_URL/videos?limit=5" | jq

print_header "测试完成!"
```

### **如何使用这个脚本**

1.  **保存文件**：
    将上面的所有代码复制到一个新文件中，命名为 `full_test.sh`，并保存在你的项目根目录下。

2.  **修改视频路径**：
    打开 `full_test.sh` 文件，找到下面这行，并**确保路径指向一个你电脑上真实存在的、较小的 `.mp4` 文件**。
    ```bash
    TEST_VIDEO_PATH="~/Downloads/test.mp4" # <--- !!! 修改这里 !!!
    ```

3.  **给予执行权限** (只需要做一次)：
    在你的终端里运行：
    ```bash
    chmod +x full_test.sh
    ```

4.  **运行测试**：
    确保你的 Go API Server 和 Worker 都在**各自的终端里正常运行**。然后，在第三个终端里，执行脚本：
    ```bash
    ./full_test.sh
    ```

### **脚本执行时的输出解读**

脚本会一步一步地执行，并打印出清晰的标题和它正在执行的命令。

-   它会先**登录**，并将完整的登录响应（包含 Token）保存到你指定的文件 `login_response.json` 中。
-   然后它会**自动提取** Token 和新创建的 Video ID，用于后续的请求。
-   在触发转码后，它会**暂停 20 秒** (`sleep 20`)。这个时间是为了给你的 Worker 留出足够的时间去完成下载、转码和上传。如果你的测试视频很大或者电脑比较慢，你可以适当增加这个时间。
-   最后，它会**调用查询接口**，并将结果用 `jq` 美化后打印在屏幕上，同时给出**检查点提示**，告诉你应该关注哪些字段的值是否正确。

这个脚本就像一个自动化测试工程师，帮你完整地验证了整个后端的核心链路。