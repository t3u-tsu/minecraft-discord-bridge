package main

import (
	"database/sql"
	"errors"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db *sql.DB
}

func InitDB(path string) (*Database, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	// テーブルの作成
	query := `
	CREATE TABLE IF NOT EXISTS invitations (
		token TEXT PRIMARY KEY,
		target_mc_server TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS links (
		guild_id TEXT PRIMARY KEY,
		target_mc_server TEXT NOT NULL,
		linked_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err = db.Exec(query)
	if err != nil {
		return nil, err
	}

	return &Database{db: db}, nil
}

// 招待トークンの作成
func (d *Database) CreateInvitation(token, targetServer string) error {
	_, err := d.db.Exec("INSERT INTO invitations (token, target_mc_server) VALUES (?, ?)", token, targetServer)
	return err
}

// トークンを使用して Discord サーバーをリンク
func (d *Database) LinkGuild(guildID, token string) (string, error) {
	var targetServer string
	err := d.db.QueryRow("SELECT target_mc_server FROM invitations WHERE token = ?", token).Scan(&targetServer)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("invalid or expired token")
		}
		return "", err
	}

	// リンクの作成（既に存在すれば上書き）
	_, err = d.db.Exec("INSERT OR REPLACE INTO links (guild_id, target_mc_server) VALUES (?, ?)", guildID, targetServer)
	if err != nil {
		return "", err
	}

	// 使用済みトークンの削除
	_, _ = d.db.Exec("DELETE FROM invitations WHERE token = ?", token)

	return targetServer, nil
}

// Discord サーバー ID から操作対象のマイクラサーバーを取得
func (d *Database) GetTargetServer(guildID string) (string, error) {
	var targetServer string
	err := d.db.QueryRow("SELECT target_mc_server FROM links WHERE guild_id = ?", guildID).Scan(&targetServer)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("this guild is not linked to any Minecraft server")
		}
		return "", err
	}
	return targetServer, nil
}

func (d *Database) Close() {
	d.db.Close()
}
