---
title: 技術選型評估與權衡分析 (Tech Stack Tradeoffs v1.0 歷史存檔)
type: planning
created: 2026-08-18
updated: 2026-08-26
status: archived
tags: [planning, tech-stack, golang, python, tradeoffs, legacy, v1]
aliases: [Tech Stack Tradeoffs v1, 技術選型決策 v1]
---

# ⚖️ 技術選型評估與權衡分析 (Tech Stack Tradeoffs v1.0)

> [!NOTE]
> **⚡ 30 秒核心精華 (Key Takeaway)**
> 經過全方位考量，`agent-observer` 選擇採用 **Pure Golang** 作為主力實作語言，以實現單一靜態二進制檔分發、極致輕量（< 15MB RSS）與無任何 Python 虛擬環境依賴的使用者體驗。

---

## 📊 一、Golang vs. Python 多維度客觀對比矩陣

| 評估維度 | 🐹 **Golang (最終選型 - 專注工具分發與低資源開銷)** | 🐍 **Python (備選方案 - 專注演算法實驗與模型微調)** |
| :--- | :--- | :--- |
| **執行檔分發體驗** | 🟢 **單一靜態 Binary**，開箱即用，0 依賴。 | 🔴 需要安裝 Python、管理 `venv`、解決 pip/C++ 編譯衝突。 |
| **記憶體佔用 (RSS)** | 🟢 **< 15 MB RSS**，極致輕量，長駐背景無感。 | 🔴 50 MB ~ 150 MB+（FastAPI/Uvicorn 基礎框架開銷）。 |
| **高並發連線模型** | 🟢 原生 Goroutine + Channel，高並發 SSE 廣播天然優勢。 | 🟡 Asyncio 事件循環，多線程受限於 GIL 鎖定機制。 |
| **Web 前端資源嵌入** | 🟢 原生支援 `//go:embed`，HTML/CSS 直接編譯進單一執行檔。 | 🔴 需要額外配置靜態資源伺服器或打包外部工具。 |
| **AI/ML 生態支援** | 🟡 依賴原生移植庫（如 `pkoukk/tiktoken-go`）。 | 🟢 官方第一方頂級支援（PyTorch, HuggingFace, Transformers）。 |

---

## 💡 二、決策邏輯與落地結論

* **為什麼 `agent-observer` 堅決選 Go？**
  * `agent-observer` 是一個給開發者長駐在本地終端機背景的 **監聽工具 (Observer Utility)**。
  * 開發者的第一訴求是「不要吃我的開發機記憶體、不要叫我安裝 Python 環境、單一指令直接跑」。
  * 因此，Golang 是兼顧 **極致效能、單一二進制檔分發與嵌入式 Web UI** 的唯一最佳解！
