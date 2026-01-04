# Minecraft Discord Bridge

NixOS 運用に最適化された、Go 製の汎用 Minecraft Discord 連携ツールです。
「この Discord サーバーには、このマイクラ鯖の操作権限をあげる」といったマルチテナント管理が可能です。

## 主な機能

- **マルチテナント管理**: 招待トークン方式で、複数の Discord サーバーから特定のマイクラ鯖を安全に管理。
- **ホワイトリスト操作**: Discord スラッシュコマンド (`/whitelist add|remove|list`)。
- **UUID 解決**: Mojang API によるプレイヤー名と UUID の自動検証。
- **管理用 Unix ソケット**: サーバーローカルから直接 RCON 操作やトークン発行が可能。
- **柔軟な RCON 接続**: TCP だけでなく、Unix ドメインソケット経由の RCON にも対応。

---

## チュートリアル: 導入手順

### 1. Discord Bot の作成
1. [Discord Developer Portal](https://discord.com/developers/applications) にアクセス。
2. **New Application** を作成し、左メニューの **Bot** を選択。
3. **Reset Token** を押してトークンを取得（後で設定ファイルに使います）。
4. **Privileged Gateway Intents** セクションで `Message Content Intent` を ON にします。

### 2. Bot の招待
1. **OAuth2 -> URL Generator** を開きます。
2. **Scopes** で `bot` と `applications.commands` をチェック。
3. **Bot Permissions** で `Send Messages`, `Read Message History` をチェック。
4. 生成された URL をブラウザで開き、自分の Discord サーバーに招待します。

### 3. アプリのビルドと設定
```bash
# ビルド
go build -o bridge .

# 設定ファイルの作成
cp config.toml.sample config.toml
# config.toml を編集してトークンやマイクラ鯖の RCON 情報を入力
```

### 4. サーバーの紐付け
1. アプリを起動: `./bridge -c config.toml`
2. 管理サーバー上で招待トークンを発行:
   `echo "invite-create <server_name>" | nc -U /run/bridge.sock`
3. Discord 側でリンクを実行:
   `/bridge-link token:<発行されたトークン>`

---

## 設定 (Configuration)

### config.toml
各マイクラサーバーの RCON 接続情報を記述します。

### 環境変数 (Environment Variables)
機密情報は環境変数で上書き可能です（NixOS や Docker での運用に推奨）。
- `DISCORD_TOKEN`: Bot トークン
- `DISCORD_ADMIN_GUILD_ID`: 管理コマンドを許可するサーバー ID
- `RCON_PASS_<server_name>`: 各サーバーの RCON パスワード

---

## コマンド一覧

### Discord スラッシュコマンド
- `/invite-create`: [管理サーバー限定] 招待トークンを発行。
- `/bridge-link`: サーバーとマイクラ鯖を紐付け。
- `/whitelist add|remove|list`: ホワイトリスト管理。

### Unix ソケットコマンド
- `invite-create <server>`: トークンを発行。
- `status`: 稼働状況の確認。
- `whitelist <server> <add|remove|list> [user]`: 直接 RCON 操作。

## ライセンス

0BSD (BSD Zero Clause License) - 詳細は `LICENSE` ファイルを参照。
