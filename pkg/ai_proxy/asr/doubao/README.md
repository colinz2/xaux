# 豆包双向流式 ASR

本目录基于火山引擎 Go 示例适配为 xaux 的 `x.ISession` 实现，不再是独立的音频文件示例程序。

协议参考：
https://docs.volcengine.com/docs/doubao-voice/bidirectional-streaming-automatic-speech-recognition-websocket?lang=zh

## 启动

在仓库根目录编译并启动：

```bash
go build -o proxy ./cmd/proxy
export DOUBAO_API_KEY=<api-key>
./proxy --asr_provider=doubao
```

只需 API Key，无需 AK/SK。WebSocket 请求会携带 `X-Api-Key`、`X-Api-Resource-Id` 和 UUID 格式的 `X-Api-Request-Id`。

`DOUBAO_WS_URL` 和 `DOUBAO_RESOURCE_ID` 可用于覆盖默认接入地址及资源 ID。默认 `volc.seedasr.sauc.duration` 为豆包流式语音识别模型 2.0 小时版；并发版使用 `volc.seedasr.sauc.concurrent`。

请求默认开启二遍识别和分句返回，使用 `result_type=full`、`end_window_size=800`，与当前累计分句及最终结果去重逻辑匹配。

## 协议适配

- 将 xaux 的 UDP PCM 音频流转换为豆包二进制音频帧。
- 最后一个音频帧使用负序号，然后等待服务端最终响应。
- 非确定的 `utterances` 映射为 xaux `SS`/`R` 消息。
- `definite=true` 的分句映射为 xaux `SE` 消息，并去除重复最终结果。
- 仅使用 `X-Api-Key` 鉴权，不读取 App Key、Access Key 或 AK/SK。
