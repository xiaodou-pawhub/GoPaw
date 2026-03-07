<p align="center">
  <img src="assets/logo.png" width="80" alt="GoPaw Logo" />
  <h1 align="center">GoPaw</h1>
</p>

<p align="center">
  <a href="https://github.com/xiaodou-pawhub/GoPaw/releases"><img src="https://img.shields.io/github/v/release/xiaodou-pawhub/GoPaw?style=flat-square" alt="Release"></a>
  <a href="https://github.com/xiaodou-pawhub/GoPaw/actions"><img src="https://img.shields.io/github/actions/workflow/status/xiaodou-pawhub/GoPaw/release.yml?branch=main&style=flat-square" alt="Build Status"></a>
  <a href="https://github.com/xiaodou-pawhub/GoPaw/blob/main/LICENSE"><img src="https://img.shields.io/github/license/xiaodou-pawhub/GoPaw?style=flat-square" alt="License"></a>
  <a href="https://github.com/xiaodou-pawhub/GoPaw/releases"><img src="https://img.shields.io/github/downloads/xiaodou-pawhub/GoPaw/latest/total?style=flat-square" alt="Downloads"></a>
  <a href="https://golang.org/doc/devel/release.html#go1.22"><img src="https://img.shields.io/github/go-mod/go-version/xiaodou-pawhub/GoPaw?style=flat-square" alt="Go Version"></a>
</p>

<p align="center">
  <a href="README.md">🇨🇳 中文</a> · 
  <a href="README.en.md">🇺🇸 English</a> · 
  <a href="README.ja.md">🇯🇵 日本語</a>
</p>

---

## 🐾 軽量 AI アシスタントワークベンチ

**GoPaw** は、Go 言語で実装された軽量でプラグイン対応のパーソナル AI アシスタントワークベンチです。ReAct 推論ループ、マルチチャンネル統合、3 層スキルシステムにより、专属の AI アシスタントを簡単に構築できます。

### コアアドバンテージ

| 特徴 | 説明 |
|------|------|
| 🚀 **超軽量** | メモリ使用量 < 150MB、単一バイナリ、ブラウザ不要 |
| 🔌 **真のプラグインアーキテクチャ** | チャンネル、ツール、スキルはすべてプラグイン化、オンデマンドロード |
| 🖥️ **サーバーフレンドリー** | Docker ワンクリックデプロイ、GUI 不要 |
| 🎯 **低参入障壁** | 一般ユーザーはコーディング不要、開発者は自由に拡張可能 |

---

## ✨ 主要機能

### 🧠 ReAct エージェント

ReAct（Reasoning + Acting）推論ループに基づく：
- **Thought-Action-Observation** 循環推論
- **マルチツール呼び出し** - ファイル操作、Shell、Web 検索、HTTP リクエスト
- **コンテキスト認識** - 会話履歴とメモリの自動ロード

### 📺 マルチチャンネル統合

| チャンネル | 説明 | 設定方法 |
|-----------|------|---------|
| **Feishu/Lark** | 企業 IM、グループ/プライベートチャット | Web UI: AppID/Secret |
| **DingTalk** | 企業 IM、グループ/プライベートチャット | Web UI: ClientID/Secret |
| **Web Console** | 内蔵 Web コンソール | http://localhost:8088 にアクセス |
| **Webhook** | 標準 HTTP インターフェース | カスタムコールバック URL 対応 |

### 🎨 3 層スキルシステム

```
Level 1: プロンプトスキル（ノーコード）
  └─ manifest.yaml + prompt.md
  └─ プロンプト注入で機能拡張

Level 2: 設定スキル（ローコード）
  └─ workflow.yaml でマルチステップタスクをオーケストレーション
  └─ 既存のツールを組み合わせる

Level 3: コードスキル（フルコード）
  └─ skill.go でカスタムツールを実装
  └─ 実行ロジックを完全に制御
```

### 🛠️ 内蔵ツールセット

| ツール | 機能 | 使用例 |
|-------|------|-------|
| `file_read` / `file_write` | ファイル操作 | 設定読み込み、ログ書き込み |
| `shell_execute` | Shell コマンド実行 | スクリプト実行、システム管理 |
| `web_search` | Web 検索（Tavily） | リアルタイム情報検索 |
| `http_get` / `http_post` | HTTP リクエスト | 外部 API 呼び出し |

### ⏰ 定期タスク

- **Cron 式スケジューリング** - 秒単位での精度
- **アクティブ時間ウィンドウ** - 迷惑時間帯を回避
- **隔離セッション** - メイン会話履歴を汚染しない

### 💾 永続メモリ

- **SQLite + FTS5** - フルテキスト検索対応
- **コンテキスト圧縮** - 過去の会話を自動要約
- **長期メモリアーカイブ** - 定期的な圧縮保存

### 🔧 ホット設定リロード

- **config.yaml** - 変更時に自動リロード
- **AGENT.md** - システムプロンプトは即時有効
- **スキル管理** - Web UI で動的に有効/無効化

---

## 🚀 クイックスタート

### Docker デプロイ（推奨）

```bash
# 1. 設定ファイルを準備
cp config.yaml.example config.yaml

# 2. サービス開始
docker compose up -d

# 3. Web UI にアクセス
open http://localhost:8088
```

> 💡 **ヒント**: 初回起動後、Web UI → 設定 → LLM プロバイダーで API キーを設定してください。設定ファイルの修正は不要です。

### ローカル開発

```bash
# 前提条件：Go 1.22+、Node.js 18+、pnpm
git clone https://github.com/xiaodou-pawhub/GoPaw.git && cd gopaw

# 依存関係インストール
go mod download && make web-install

# 設定初期化
go run ./cmd/gopaw init

# 開発サーバー起動
make dev
```

アクセス：
- **フロントエンド（HMR ホットリロード）**: http://localhost:5173
- **バックエンド API**: http://localhost:8088

### 本番モード（単一バイナリ）

```bash
# ビルド
make build

# 初期化して開始
./gopaw init
./gopaw start
```

---

## 📚 ドキュメント

| ドキュメント | 説明 |
|-------------|------|
| [デプロイガイド](docker/DEPLOY.md) | Docker デプロイ、サーバー設定、运维コマンド |
| [スキル開発](skills/SKILLS.md) | カスタムスキルの作成、プロンプト作成ガイド |
| [プラグイン仕様](GoPaw_Design.md#10-プラグイン仕様) | チャンネルプラグイン、ツールプラグインの開発 |
| [API リファレンス](#rest-api) | REST API、WebSocket インターフェース |

---

## 🔧 CLI コマンド

| コマンド | 説明 |
|---------|------|
| `gopaw init` | デフォルトの config.yaml を生成 |
| `gopaw start [--config path]` | サービス開始 |
| `gopaw version` | バージョン情報表示 |

---

## 📁 プロジェクト構成

```
gopaw/
├── cmd/gopaw/         # プログラムエントリー
├── internal/          # コアビジネスロジック
│   ├── agent/         # ReAct エンジン
│   ├── memory/        # メモリシステム（SQLite + FTS5）
│   ├── channel/       # チャンネル管理
│   ├── skill/         # スキルローダー
│   ├── tool/          # ツールレジストリ
│   ├── llm/           # LLM クライアント
│   ├── scheduler/     # Cron スケジューラー
│   ├── server/        # HTTP/WebSocket サービス
│   ├── config/        # 設定管理
│   ├── platform/      # 内蔵チャンネルプラグイン
│   └── tools/         # 内蔵ツール
├── pkg/               # 公開インターフェース（プラグイン開発者向け）
│   ├── plugin/        # ChannelPlugin / Tool / Skill インターフェース
│   └── types/         # 統一メッセージタイプ
└── skills/            # ユーザー定義スキルディレクトリ
```

---

## 🌐 REST API

| エンドポイント | メソッド | 説明 |
|--------------|---------|------|
| `/api/agent/chat` | POST | メッセージ送信 |
| `/api/agent/chat/stream` | GET | SSE ストリーミングレスポンス |
| `/api/agent/sessions` | GET | 全セッション一覧 |
| `/api/skills` | GET/PUT | スキル管理 |
| `/api/channels/health` | GET | チャンネルヘルスステータス |
| `/api/cron` | GET/POST | 定期タスク管理 |
| `/api/system/version` | GET | バージョン情報 |
| `/health` | GET | ヘルスチェック |
| `/ws` | WS | WebSocket 双方向通信 |

---

## 🧩 プラグイン開発

### チャンネルプラグインの開発

```go
package myplugin

import "github.com/xiaodou-pawhub/GoPaw/internal/channel"

type MyPlugin struct{}

func (p *MyPlugin) Name() string { return "my_channel" }
// ... すべてのインターフェースメソッドを実装

func init() {
    channel.Register(&MyPlugin{})
}
```

### ツールプラグインの開発

```go
package mytools

import "github.com/xiaodou-pawhub/GoPaw/internal/tool"

type MyTool struct{}

func (t *MyTool) Name() string { return "my_tool" }
// ... すべてのインターフェースメソッドを実装

func init() {
    tool.Register(&MyTool{})
}
```

### スキルの作成

`skills/` ディレクトリにサブディレクトリを作成：

```yaml
# skills/my_skill/manifest.yaml
name: my_skill
version: 1.0.0
display_name: マイスキル
level: 1  # 1=プロンプト / 2=設定 / 3=コード
```

```markdown
<!-- skills/my_skill/prompt.md -->
## マイスキルの機能説明
ユーザーが...と質問したとき、あなたは...するべきです
```

---

## 🛠️ ビルドコマンド

```bash
make dev            # 開発モード：Vite HMR + Go バックエンド
make build          # 本番モード：フロントエンド圧縮 + 埋め込み Go バイナリ
make test           # 全テスト実行
make docker-build   # Docker イメージビルド
make lint           # コードリンティング
```

---

## 📋 システム要件

| 環境 | 要件 |
|------|------|
| **本番環境** | Docker または Go 1.22+ ランタイム |
| **最小メモリ** | 50MB（LLM API レスポンスを除く） |
| **ストレージ** | SQLite データベース（デフォルト `~/.gopaw/gopaw.db`） |
| **ネットワーク** | LLM API へのアクセス（OpenAI または互換） |

---

## 📄 ライセンス

AGPL-3.0

<p align="center">
  Designed with ❤️ by <a href="https://github.com/xiaodou997">xiaodou997</a>
</p>
