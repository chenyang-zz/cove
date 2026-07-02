// Package documentparse 提供业务无关的文档二进制文本提取能力。
//
// 包内默认支持 txt、markdown、html、docx 和 pdf。调用方可以通过 WithExtractor
// 替换任意格式的提取器，也可以通过 WithTextDecoder 替换纯文本解码器。
package documentparse
