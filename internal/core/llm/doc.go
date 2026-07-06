// Package llm 定义业务无关的模型客户端接口、消息结构和调用选项。
//
// 本包保留简单文本调用接口，同时通过 Client.InvokeResult 提供结构化生成结果，供需要
// token 用量、停止原因或工具调用信息的上层模块按需使用。
package llm
