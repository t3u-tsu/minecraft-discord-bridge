# Minecraft Discord Bridge

NixOS 運用に最適化された、Go 製の汎用 Minecraft Discord 連携ツールです。
「この Discord サーバーには、このマイクラ鯖の操作権限をあげる」といったマルチテナント管理が可能です。

## 主な機能

- **マルチテナント管理**: 招待トークン方式で、複数の Discord サーバーから特定のマイクラ鯖を安全に管理。
- **名前付き招待**: どのトークンを誰に渡したか名前を付けて管理可能。
- **ロールベースの権限管理**: リンク時に「管理ロール」を指定し、そのロール保持者のみがコマンドを実行可能。
- **参加表明システム**: `/join` コマンドで、一般ユーザーに管理ロールを自動付与。
- **ホワイトリスト操作**: Discord スラッシュコマンド (`/whitelist add|remove|list`)。
- **UUID 解決**: Mojang API によるプレイヤー名と UUID の自動検証。
- **管理用 Unix ソケット**: サーバーローカルから直接 RCON 操作やトークン管理が可能。
- **自動マイグレーション**: データベースのスキーマ変更を起動時に自動適用。

---

## チュートリアル: 導入手順

### 1. Discord Bot の作成
1. [Discord Developer Portal](https://discord.com/developers/applications) にアクセス。
2. **New Application** を作成し、左メニューの **Bot** を選択。
3. **Reset Token** を押してトークンを取得。
4. **Privileged Gateway Intents** で `Server Members Intent` を ON にします（ロール付与に必要）。

### 2. Bot の招待
1. **OAuth2 -> URL Generator** を開きます。
2. **Scopes** で `bot` と `applications.commands` をチェック。
3. **Bot Permissions** で `Manage Roles`, `Send Messages` をチェック。
4. 生成された URL で自分のサーバーに招待。

### 3. 設定と起動
`config.toml` に RCON 情報を記述し、バイナリを起動します。

### 4. サーバーの紐付け
1. 管理サーバー（またはソケット）でトークンを発行:
   `/token create server:nitac23s name:管理チームA`
2. 管理を任せたいサーバーでリンクを実行:
   `/bridge-link token:<トークン> role:@管理者ロール`
3. 以降、そのサーバーのユーザーは `/join` でロールを受け取り、マイクラを管理できます。

---

## コマンド一覧

### Discord スラッシュコマンド
- `/token create <srv> <name>`: [管理サーバー限定] 名前付きトークンを発行。
- `/token list`: [管理サーバー限定] 有効なトークンを一覧表示。
- `/token revoke <token>`: [管理サーバー限定] トークンを失効させる。
- `/bridge-link <token> <role>`: Discord サーバーとマイクラ鯖を紐付け、管理ロールを指定。
- `/join`: 設定された管理ロールを取得。
- `/whitelist <add|remove|list>`: ホワイトリストを操作。

### Unix ソケットコマンド
- `invite-create <srv> <name>`: トークン発行。
- `invite-list`: トークン一覧。
- `invite-revoke <token>`: トークン失効。
- `status`: 稼働状況確認。
- `whitelist <srv> <add|remove|list> [user]`: 直接 RCON 操作。

## ライセンス

0BSD (BSD Zero Clause License)
