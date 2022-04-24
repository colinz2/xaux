## 采集工具

- 声音采集工具做采样率转换
- 通过 UDP 发送声音

## agent

- agent 接收语音，调用阿里云识别
- 语音话单保存
- 语义识别
- 问答检索
- 前端展示接口

## proxy

- 运行在 linux
- 语音话单保存
- 支持多种公有云调用

# protocol

## 信令

tcp 长连接

### 开始

request

```json
{
  "cmd": "start",
  "config": {
    "sampleRate": 48000,
    "bytesPerSample": 2
  }
}
```

response

```json
{
  "cmd": "start",
  "sessionID": 1,
  "udpPort": 11024
}
```

### 结束

请求

```json
{
  "cmd": "end"
}
```

响应

```json
{
  "cmd": "end",
  "msg": "closed"
}
```

### 识别结果

```json
{
  "cmd": "recognize",
  "result": {
    
  }
}
```

## 语音流

| 0-3 | 4-7 | 8-15 | | sessionID | sequence | reserve |

## 命名

sober coffee

