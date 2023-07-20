# mcache

## 介绍
`mcache` 是一个内存池，它在 `sync.Pool` 中保留内存以提高 malloc 的性能。
用法很简单：直接调用 `mcache.Malloc`，别忘了用 `Free` 释放它！