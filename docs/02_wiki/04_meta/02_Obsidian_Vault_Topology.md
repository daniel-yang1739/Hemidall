---
title: 現代化 Obsidian 知識庫拓撲架構、目錄職責與專案協同憲法
type: architecture
created: 2026-08-26
updated: 2026-08-26
status: completed
tags: [meta, obsidian-vault, vault-topology, directory-mandates, information-architecture, repository-constitution, km-system]
aliases: [Vault Topology, Obsidian 知識庫拓撲, 目錄職責架構, 專案協同憲法, Wiki 終點論]
---

# 💎 現代化 Obsidian 知識庫拓撲架構、目錄職責與專案協同憲法

> [!NOTE]
> **⚡ 30 秒核心精華 (Key Takeaway)**
> 現代軟體工程與 AI 協同開發中，知識管理庫（Obsidian Vault）常因「混雜文章草稿」、「隨機編號混亂」與「缺乏清晰的目錄職責界線」而退化為難以維護的技術垃圾場。
> 本架構提出了三大治理體系：**「全域 ASCII 檔案結構樹與 High-Level 目錄職責矩陣」**、**「Wiki 終點論與三權分立拓撲」** 與 **「.obsidian 嚴格白名單版本控制 + AGENTS.md 語言憲法」**，保障知識資產的長期複利性與團隊協同的一致性。

---

## 🌲 一、全域檔案拓撲結構樹 (Global Vault Tree)

知識庫採用模組化、正交化（MECE）且語意自洽的目錄樹設計：

```text
docs/
├── 01_raw/                   # 📥 [素材收集池] 臨時原始資料池 (遵循 Digest & Delete 原則，提煉後即刪除)
│   └── README.md
├── 02_wiki/                  # 🧠 [核心資產庫] 永久長效知識中樞 (面向人類好讀好學、極致精煉、嚴密編號)
│   ├── 01_theory/            #    ⚡ [推論物理] Attention 數學、KV Cache 顯存大小、Prompt Caching 物理時序、雙水位線壓縮
│   │   ├── 01_Transformer_Prefill_vs_Decode.md
│   │   ├── 02_KV_Cache_Mechanics.md
│   │   ├── 03_Prompt_Caching_Lifecycle.md
│   │   ├── 04_Context_Compaction_and_Summarization.md
│   │   └── index.md
│   ├── 02_architecture/      #    🏛️ [系統落地] Context 5 維度、LCP 快取演算法、雙層 SQLite 狀態機、雙軌遙測、TUI 盒模型、會話快切
│   │   ├── 01_Context_5_Dimensions.md
│   │   ├── 02_Token_Calculation_and_LCP.md
│   │   ├── 03_Agent_Storage_and_State_Machine.md
│   │   ├── 04_Service_Plan_Agent_Observer.md
│   │   ├── 05_Model_Payload_and_API_Traces.md
│   │   ├── 06_Dual_Track_Telemetry_and_Window_Accounting.md
│   │   ├── 07_TUI_Engine_and_Terminal_Layout_Mechanics.md
│   │   ├── 08_Interactive_Session_Switching_and_Anti_Jitter.md
│   │   └── index.md
│   ├── 03_planning/          #    🏆 [系列藍圖] v1/ (初版存檔) 與 v2/ (萬能觀測與極致壓縮旗艦版)
│   │   ├── v1/               #       📜 [初版存檔] 01~04 系列企劃與 30 天大綱歷史存檔
│   │   ├── v2/               #       🚀 [旗艦主線] 01~04 萬能觀測中樞、AI 基礎課與時代終章 (v2.0)
│   │   └── index.md          #       🧭 規劃版本演進總導覽 (Planning MOC)
│   ├── 04_meta/              #    🤖 [協同工程] 19 位審查官雙輪對抗審查、Obsidian 拓撲與協同憲法
│   │   ├── 01_Multi_Agent_Adversarial_Review_Pattern.md
│   │   ├── 02_Obsidian_Vault_Topology.md
│   │   └── index.md
│   ├── 05_troubleshooting/   #    🛠️ [實戰手冊] SRE 四段式故障覆盤、QA 問答與 Runbook 診斷 SOP
│   │   ├── 01_Context_Inflation_and_Intermediate_Compounding.md
│   │   ├── 02_Startup_Warmup_Double_Ingestion_and_Cache_Lag.md
│   │   ├── 03_TUI_ANSI_Escape_Truncation_and_Overscroll_Lag.md
│   │   └── index.md
│   └── index.md              #    🧭 Wiki 根目錄全景導覽 (Wiki Root MOC)
├── ithome_draft/             # ✍️ [專案工作區] 鐵人賽 30 天文章草稿撰寫區與寫作進度看板
│   └── README.md
├── ithome_ready/             # 🚀 [發布定稿區] 完稿並排版完畢、可直接複製發布至 iThome 的定稿庫
│   └── README.md
├── reviews/                  # 📜 [審查報告室] 19 角色雙輪深層對抗審查報告書、思維鏈挑惕紀錄與終審簽核
│   ├── 2026-08-26_02-15-00_two_round_audit_report.md
│   ├── 2026-08-26_02-28-53_meta_module_audit_report.md
│   ├── 2026-08-26_15-11-37_concrete_walkthrough_audit_report.md
│   ├── 2026-08-26_23-13-21_wiki_distillation_comprehensive_audit_report.md
│   └── 2026-08-27_02-15-00_multi_agent_adversarial_review_audit_report.md
├── schema.md                 # 📐 [知識庫憲法] Obsidian 拓撲原則、三權分立與卡片規範
├── log.md                    # ⏱️ [時序變更日誌] 全庫 Append-Only 操作與提煉紀錄
└── index.md                  # 🗺️ [全域總導覽] 最高導覽中樞 (Global MOC)
```

---

## 🏛️ 二、High-Level 頂層目錄職責與生命週期矩陣 (Directory Mandates)

每個 High-Level 目錄均被賦予不可混淆的職責與生命週期治理規則：

| 目錄路徑 | 核心職責與功能定位 | 生命週期與治理規則 (Lifecycle Policy) | 存儲內容範例 |
| :--- | :--- | :--- | :--- |
| **`01_raw/`** | **臨時素材收集池 (Intake Pool)**<br/>存放未加工的 API Traces、逆向日誌、臨時截圖與靈感碎片。 | **Digest & Delete**：一旦經 `wiki-distiller` 100% 提煉進 `02_wiki/`，原始檔案立即安全清理刪除，保持素材池極致乾淨。 | 原始 JSONL 日誌、API payload 截圖、發想草案。 |
| **`02_wiki/`** | **永久核心知識資產庫 (Permanent Asset Hub)**<br/>世界級、排版精美、結構自洽、具備 4 維度圖解導讀與極簡演繹實例的永久資產。 | **永不刪除 / 持續迭代**：嚴禁存放未完成的草稿。依大腦認知演進（`01_` $\to$ `02_` $\to$ `03_` $\to$ `04_` $\to$ `05_`）嚴密編號。 | 20 篇經過 19 角色雙輪審查的長效技術卡片與排查 Runbook。 |
| **`ithome_draft/`** | **文章草稿工作區 (Writing Workspace)**<br/>以 Wiki 為武器庫，專門用於撰寫 iThome 鐵人賽 30 天連載草稿。 | **短期專案週期**：與底層 Wiki 完全解耦，專注於文章受眾節奏、開場 Hook 與章節編排。 | Day 01 ~ Day 30 連載文章草稿。 |
| **`ithome_ready/`** | **定稿發布庫 (Production Release)**<br/>完成最終潤稿、排版校對，隨時可直接複製 Po 到發文後台。 | **發布就緒**：代表可對外公開發表的正式文章。 | 排版完畢的最終發布 Markdown。 |
| **`reviews/`** | **審查辯論與終審報告室 (Audit Room)**<br/>存放 19 位頂尖審查員的深層思維鏈挑惕、Main Agent 駁回/採納辯論與大檢察官簽核。 | **歷史審計存檔**：永久留存審查軌跡，正文 0 人名，所有審查員人名與辯論完整留存於此。 | 各模組雙輪審查全景報告書。 |

---

## 🏛️ 三、三權分立拓撲架構圖與精讀指引

傳統專案常將筆記庫與文章輸出混在一起，導致文檔變成粗糙的中間站。本架構確立了三權分立公理：

```mermaid
flowchart TD
    subgraph RawPool ["1. 📥 01_raw/ (素材收集池)"]
        R["未消化的原始日誌、API Traces、想法草案<br/>⚡ 遵循 Digest & Delete 原則 (100% 提煉後即刻刪除)"]
    end

    subgraph PermanentWiki ["2. 🧠 02_wiki/ (終極知識資產庫) - 核心中樞"]
        direction TB
        W1["⚡ 01_theory/<br/>(推論物理與數學模型 - 認知起點)"] --> W2["🏛️ 02_architecture/<br/>(通用系統與演算法模式 - 系統落地)"]
        W2 --> W3["🏆 03_planning/<br/>(系列藍圖與規格規劃 - 產品全景)"]
        W3 --> W4["🤖 04_meta/<br/>(AI 協同工程與知識庫方法論)"]
        W2 --> W5["🛠️ 05_troubleshooting/<br/>(實戰故障排查與 Runbook 手冊)"]
    end

    subgraph OutputProject ["3. ✍️ 專案輸出空間 (Writing Workspace)"]
        direction TB
        D["ithome_draft/<br/>(文章草稿與進度看板)"] --> P["ithome_ready/<br/>(定稿直接複製發布區)"]
    end

    R -->|極致深度提煉 (wiki-distiller)| PermanentWiki
    PermanentWiki -->|精準調用知識武庫| D
    
    style RawPool fill:#fdf6e2,stroke:#d33682
    style PermanentWiki fill:#d1ecf1,stroke:#0c5460
    style OutputProject fill:#d4edda,stroke:#155724
```

### 📖 圖表深度精讀指南 (Diagram Walkthrough)

1. **【核心視野】**：本圖展示了資訊從「Raw 原始雜訊」流入「Wiki 長效資產」，再賦能於「短期寫作專案」的單向高熵轉低熵流程。
2. **【看圖路徑 (Step-by-Step)】**：
   * **步驟 1 (素材池 01_raw/)**：收集暫存日誌。透過 `wiki-distiller` 提煉完成後，**原始檔案立即刪除**，保持輸入池乾淨。
   * **步驟 2 (知識庫 02_wiki/)**：這是知識庫的核心終點（Destination）。內部劃分為 5 個模組，彼此以 `[[WikiLinks]]` 緊密互連，形成無孤島知識圖譜。
   * **步驟 3 (專案區 ithome_*)**：寫作專案從 Wiki 汲取結構化的素材與深度見解進行輸出，專案結束後，Wiki 本身不受任何污染。

---

## 🔒 四、.obsidian 白名單版本控制與 Git 協同體系

為避免 Obsidian 自動生成的快取、工作區狀態與熱加載檔案造成嚴重的 Git 衝突，本專案在根目錄 `.gitignore` 建立了**嚴格白名單制度**：

```gitignore
# 1. 預設全面忽略所有 Obsidian 內部運行時與快取檔案
docs/.obsidian/*
.obsidian/*

# 2. 嚴格白名單：僅允許追蹤以下 4 個核心設定檔
!docs/.obsidian/app.json                  # 編輯器核心設置、全寬模式
!docs/.obsidian/appearance.json           # 主題選擇器與字體設定
!docs/.obsidian/core-plugins.json         # 啟用的官方核心插件
!docs/.obsidian/community-plugins.json    # 啟用的社群插件清單
```

---

## 📜 五、Repository Constitution (專案協同憲法)

本知識庫與代碼庫受根目錄 [`AGENTS.md`](file:///Users/daniel_y_yang/Documents/self/ithome2026/AGENTS.md) 專案憲法約束：

1. **語言鐵律 (Language Policy)**：
   * `agent-observer/` 代碼、註解、CLI 輸出與測試 **100% 必須為英文**。
   * `docs/` 知識庫與文章 **100% 必須為繁體中文 (Traditional Chinese)**。
2. **正文 0 人名污染 (Zero-Persona Contamination)**：
   * 審查官人名屬於審查室，Wiki 內文必須維持客觀、沉穩的世界級技術規格書口吻。
3. **強制圖表 4 維度深度導讀**：
   * 每張 Mermaid 圖表後必須配備【核心視野】、【看圖路徑】、【色彩符號意義】與【底層工程細節】。

---

## 🔗 六、相關概念與延伸閱讀
* [[01_Multi_Agent_Adversarial_Review_Pattern]]：19 位審查官雙輪對抗審查架構。
* [[05_troubleshooting/index|05_troubleshooting: 實戰故障排查與 Runbook 手冊]]：SRE 實戰覆盤與排查武器庫。
* [[03_planning/01_Master_Plan]]：2026 iThome 鐵人賽總策劃案。
* [[schema|知識庫規範與卡片結構書]]：Obsidian 知識庫規範書。
