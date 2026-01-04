package main

import (
	"database/sql"
	"errors"
	"log"

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

	// 初期テーブル作成
	query := `
	CREATE TABLE IF NOT EXISTS invitations (
		token TEXT PRIMARY KEY,
		target_mc_server TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS links (
		guild_id TEXT PRIMARY KEY,
		target_mc_server TEXT NOT NULL,
		management_role_id TEXT DEFAULT '',
		allowed_channel_id TEXT DEFAULT '',
		linked_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err := db.Exec(query); err != nil {
		return nil, err
	}

	// --- マイグレーションロジック ---
	// invitations テーブルに name カラムがあるかチェック
	var count int
	err = db.QueryRow("SELECT count(*) FROM pragma_table_info('invitations') WHERE name='name'").Scan(&count)
	if err == nil && count == 0 {
		log.Println("[DB] Migrating: Adding 'name' column to invitations table")
		if _, err := db.Exec("ALTER TABLE invitations ADD COLUMN name TEXT DEFAULT ''"); err != nil {
			log.Printf("[DB] Error adding column 'name': %v", err)
		}
	}

	// links テーブルに management_role_id カラムがあるかチェック
	err = db.QueryRow("SELECT count(*) FROM pragma_table_info('links') WHERE name='management_role_id'").Scan(&count)
	if err == nil && count == 0 {
		log.Println("[DB] Migrating: Adding 'management_role_id' column to links table")
		if _, err := db.Exec("ALTER TABLE links ADD COLUMN management_role_id TEXT DEFAULT ''"); err != nil {
			log.Printf("[DB] Error adding column 'management_role_id': %v", err)
		}
	}

	// links テーブルに allowed_channel_id カラムがあるかチェック
	err = db.QueryRow("SELECT count(*) FROM pragma_table_info('links') WHERE name='allowed_channel_id'").Scan(&count)
	if err == nil && count == 0 {
		log.Println("[DB] Migrating: Adding 'allowed_channel_id' column to links table")
		if _, err := db.Exec("ALTER TABLE links ADD COLUMN allowed_channel_id TEXT DEFAULT ''"); err != nil {
			log.Printf("[DB] Error adding column 'allowed_channel_id': %v", err)
		}
	}

	// --- 既存の NULL データを空文字列にクリーンアップ ---
	_, _ = db.Exec("UPDATE invitations SET name = '' WHERE name IS NULL")
	_, _ = db.Exec("UPDATE links SET management_role_id = '' WHERE management_role_id IS NULL")
	_, _ = db.Exec("UPDATE links SET allowed_channel_id = '' WHERE allowed_channel_id IS NULL")

	return &Database{db: db}, nil
}

// 招待トークンの作成 (名前付き)
func (d *Database) CreateInvitation(token, targetServer, name string) error {
	_, err := d.db.Exec("INSERT INTO invitations (token, target_mc_server, name) VALUES (?, ?, ?)", token, targetServer, name)
	return err
}

type Invitation struct {
	Token        string
	TargetServer string
	Name         string
	CreatedAt    string
}

// 招待トークンの一覧取得
func (d *Database) ListInvitations() ([]Invitation, error) {
	rows, err := d.db.Query("SELECT token, target_mc_server, name, created_at FROM invitations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Invitation
	for rows.Next() {
		var i Invitation
		if err := rows.Scan(&i.Token, &i.TargetServer, &i.Name, &i.CreatedAt); err != nil {
			continue
		}
		list = append(list, i)
	}
	return list, nil
}

// トークンの失効
func (d *Database) RevokeInvitation(token string) error {
	res, err := d.db.Exec("DELETE FROM invitations WHERE token = ?", token)
	if err != nil {
		return err
	}
	count, _ := res.RowsAffected()
	if count == 0 {
		return errors.New("invitation token not found")
	}
	return nil
}

// Discord サーバーをリンク (Role ID & Channel ID 対応)
func (d *Database) LinkGuild(guildID, token, roleID, channelID string) (string, error) {
	var targetServer string
	err := d.db.QueryRow("SELECT target_mc_server FROM invitations WHERE token = ?", token).Scan(&targetServer)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("invalid or expired token")
		}
		return "", err
	}

	// リンクの作成
	_, err = d.db.Exec("INSERT OR REPLACE INTO links (guild_id, target_mc_server, management_role_id, allowed_channel_id) VALUES (?, ?, ?, ?)", guildID, targetServer, roleID, channelID)
	if err != nil {
		return "", err
	}

	// 使用済みトークンの削除
	_, _ = d.db.Exec("DELETE FROM invitations WHERE token = ?", token)

	return targetServer, nil
}

// リンク情報の取得
func (d *Database) GetLinkInfo(guildID string) (string, string, string, error) {
	var targetServer, roleID, channelID string
	err := d.db.QueryRow("SELECT target_mc_server, management_role_id, allowed_channel_id FROM links WHERE guild_id = ?", guildID).Scan(&targetServer, &roleID, &channelID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", "", errors.New("this guild is not linked to any Minecraft server")
		}
		return "", "", "", err
	}
	return targetServer, roleID, channelID, nil
}

type LinkInfo struct {
	GuildID      string
	TargetServer string
	RoleID       string
	ChannelID    string
}

func (d *Database) ListLinks() ([]LinkInfo, error) {
	rows, err := d.db.Query("SELECT guild_id, target_mc_server, management_role_id, allowed_channel_id FROM links")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []LinkInfo
	for rows.Next() {
		var l LinkInfo
		if err := rows.Scan(&l.GuildID, &l.TargetServer, &l.RoleID, &l.ChannelID); err != nil {
			continue
		}
		list = append(list, l)
	}
	return list, nil
}

func (d *Database) Close() {
	d.db.Close()
}
