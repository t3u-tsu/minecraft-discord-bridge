# Minecraft Discord Bridge

NixOS 運用に最適化された、Go 製の汎用 Minecraft Discord 連携ツールです。複数の Discord サーバーから特定のマイクラ鯖を安全に管理できる「マルチテナント方式」を採用しています。

## 主な機能

- **マルチテナント管理**: 招待トークンを使用して、特定の Discord サーバーに特定のマイクラ鯖の操作権限を動的に付与。
- **ホワイトリスト操作**: Discord から `/whitelist <add|remove|list>` を実行可能。
- **UUID 解決**: Mojang API を介したプレイヤー名と UUID の自動検証。
- **二系統の入力**: Discord スラッシュコマンドに加え、ローカル管理用の Unix ドメインソケットを提供。
- **RCON 対応**: TCP だけでなく、Unix ドメインソケット経由の RCON 通信にも対応。

## アーキテクチャ

システムは以下の 3 つの要素で構成されます。

1.  **SQLite DB**: Discord サーバーとマイクラ鯖の紐付け、および招待トークンの管理。
2.  **TOML 設定**: マイクラ鯖の接続情報（RCON）などの静的な設定。
3.  **Unix ソケット**: サーバーローカルからの直接操作用。

## クイックスタート

### 1. ビルド
```bash
nix develop
go build -o bridge .
```

### 2. 設定
`config.toml` を作成し、Discord トークンとマイクラ鯖の情報を記述します。

### 3. 実行
```bash
./bridge -c config.toml
```

## ライセンス

0BSD (BSD Zero Clause License)
