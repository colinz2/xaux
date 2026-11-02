# xaux

## Proxy

Proxy 支持阿里云和豆包语音识别，通过 `--asr_provider` 选择；`cmd/proxy` 当前默认使用豆包：

```bash
# 豆包双向流式语音识别（当前默认）
./proxy --asr_provider=doubao

# 阿里云
./proxy --asr_provider=aliyun
```

Windows 编译：

```powershell
go build -o proxy.exe .\cmd\proxy
```

Linux 编译：

```bash
make proxy
```

### 调试与录音（默认关闭）

只有显式指定 `--debug` 才会启用详细日志和录音。正常运行不需要、也不应长期开启。

Windows：

```powershell
.\proxy.exe --asr_provider=doubao --debug
# 可选：指定录音目录
.\proxy.exe --asr_provider=doubao --debug --record_dir=.\debug_recordings
```

也可以直接通过启动脚本临时开启：

```bat
start_doubao_proxy.bat --debug
```

Linux / macOS：

```bash
./proxy --asr_provider=doubao --debug
./proxy --asr_provider=doubao --debug --record_dir=./debug_recordings
# 或使用启动脚本
./start_doubao_proxy.sh --debug
```

默认在 `recordings/<provider>_session_<id>_<timestamp>/` 下生成：

| 文件 | 内容 |
| --- | --- |
| `input.pcm` | Proxy 从客户端 UDP 收到的原始 PCM |
| `asr_16k.pcm` | 重采样后、实际成功提交给 ASR SDK/WebSocket 的单声道 16 kHz PCM |
| `session.json` | Provider、会话 ID、采样率、位深、起止时间、字节数及两路音频 RMS/峰值电平 |

PCM 均为 signed 16-bit little-endian。可用 `ffplay` 直接检查 ASR 输入：

```bash
ffplay -f s16le -ar 16000 -ac 1 asr_16k.pcm
```

检查原始输入时使用 `session.json` 中的采样率，声道数需与采集端保持一致。例如 48 kHz 单声道：

```bash
ffplay -f s16le -ar 48000 -ac 1 input.pcm
```

### Client 联调与故障定位

Client 应按以下顺序调用，并检查每一步返回的错误：

1. `NewClient` 连接 Proxy，收到成功的 `RS` 后才认为 `Start` 成功。
2. `Send` 发送符合 `StartConfig` 的 PCM；格式为 16-bit、小端，采样率使用 16000 或 48000。
3. `End` 会先发送剩余短尾包，再等待 `RE`。Start/End 中的 ASR 错误会直接作为 Go error 返回。
4. 回调中应能看到 `SS`、`R` 或 `SE`；完成后调用 `Close`。

开启 Proxy debug 后按现象判断：

| 现象 | 优先排查 |
| --- | --- |
| 没有 `RS`，或 `RS` 直接返回鉴权、建连、超时错误 | ASR 凭据、Proxy 到云服务的 DNS/TLS/WebSocket 网络；此阶段与音频无关 |
| `input.pcm` 为 0 字节 | Client 未调用 Send、StartConfig/UDP 端口错误、防火墙或 UDP 路由；检查 unknown session 日志 |
| 日志出现 sequence gap、duplicate/out-of-order | Client 发送序号或 UDP 网络存在丢包、重复、乱序 |
| 日志出现 `CLIENT AUDIO WARNING`，且 RMS 低于 -50 dBFS | Client 音频接近静音；检查麦克风、系统增益及声道选择，不要盲目修改 ASR 配置 |
| `input.pcm` 有数据但 `asr_16k.pcm` 为 0 字节 | 采样参数、重采样插件或 ASR 发送阶段；查看 resample/send 错误 |
| `input.pcm` 杂音、静音或播放速度异常 | Client 采集格式、位深、采样率、声道或字节序错误 |
| `input.pcm` 正常但 `asr_16k.pcm` 异常 | 重采样或声道配置问题 |
| 两路音频都清晰正常，但无识别响应或识别错误 | ASR 网络/协议、资源 ID、模型或服务端识别问题；结合 ASR logid 联系服务方 |

Debug 每 100 包输出一次 UDP 统计，结束时输出汇总。录音和识别文本可能包含敏感信息，只在排障时开启，并妥善清理。

### 阿里云鉴权

```bash
export ALI_AKID=<access-key-id>
export ALI_AKKEY=<access-key-secret>
export ALI_APPKEY=<app-key>
```

### 豆包鉴权

也可以直接编辑仓库根目录的启动脚本，将其中的 `your-api-key` 替换为实际 API Key：

```bat
REM Windows
start_doubao_proxy.bat
```

```bash
# Linux / macOS
chmod +x start_doubao_proxy.sh
./start_doubao_proxy.sh
```

脚本内已经包含 WebSocket 地址、Resource ID 和 API Key。豆包只需 API Key，无需配置 AK/SK。

配置 API Key：

```bash
export DOUBAO_API_KEY=<api-key>
./proxy --asr_provider=doubao
```

WebSocket 请求头包含 `X-Api-Key`、`X-Api-Resource-Id` 和 UUID 格式的 `X-Api-Request-Id`。

可选环境变量：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `DOUBAO_WS_URL` | `wss://openspeech.bytedance.com/api/v3/sauc/bigmodel_async` | WebSocket 地址 |
| `DOUBAO_RESOURCE_ID` | `volc.seedasr.sauc.duration` | `X-Api-Resource-Id` |

豆包接口接收单声道、16-bit PCM，支持 16 kHz 和 48 kHz 输入；48 kHz 输入需要 `resample.so` 插件。
