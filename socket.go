package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

// コマンド実行の核となる関数
// 引数: コマンド文字列
// 戻り値: レスポンス文字列, エラー
func ProcessCommand(input string, db *Database, cfg *Config) (string, error) {
	args := strings.Fields(input)
	if len(args) == 0 {
		return "", nil
	}

	command := args[0]
	log.Printf("[CORE] Processing command: %s", input)
	log.Printf("[CORE] Full command string to send: %q", input)

	switch command {
	case "invite-create":
		if len(args) < 3 {
			return "Usage: invite-create <server_name> <invitation_name>", nil
		}
		srvName := args[1]
		// Check if server exists in config
		if _, ok := cfg.Servers[srvName]; !ok {
			return fmt.Sprintf("Error: Server '%s' not found in configuration", srvName), nil
		}
		inviteName := strings.Join(args[2:], " ")
		token := GenerateToken(srvName)
		if err := db.CreateInvitation(token, srvName, inviteName); err != nil {
			return "", err
		}
		return fmt.Sprintf("Token Created: %s (Target: %s, Name: %s)", token, srvName, inviteName), nil

	case "invite-list":
		list, err := db.ListInvitations()
		if err != nil {
			return "", err
		}
		if len(list) == 0 {
			return "No active invitations.", nil
		}
		var sb strings.Builder
		sb.WriteString("Active Invitations:\n")
		for _, i := range list {
			sb.WriteString(fmt.Sprintf("- %s | Server: %s | Name: %s | Created: %s\n", i.Token, i.TargetServer, i.Name, i.CreatedAt))
		}
		return sb.String(), nil

	case "invite-revoke":
		if len(args) < 2 {
			return "Usage: invite-revoke <token>", nil
		}
		if err := db.RevokeInvitation(args[1]); err != nil {
			// Ensure error string is returned
			return fmt.Sprintf("Error: %v", err), nil
		}
		return "Token revoked.", nil

	case "unlink":
		if len(args) < 2 {
			return "Usage: unlink <guild_id>", nil
		}
		if err := db.UnlinkGuild(args[1]); err != nil {
			return fmt.Sprintf("Error: %v", err), nil
		}
		return "Guild unlinked.", nil

	case "links":
		list, err := db.ListLinks()
		if err != nil {
			return "", err
		}
		if len(list) == 0 {
			return "No guilds linked.", nil
		}
		var sb strings.Builder
		sb.WriteString("Active Links:\n")
		for _, l := range list {
			sb.WriteString(fmt.Sprintf("- Guild: %s | Server: %s | Role: %s | Channel: %s\n", l.GuildID, l.TargetServer, l.RoleID, l.ChannelID))
		}
		return sb.String(), nil

	case "whitelist":
		// whitelist <server_name> <add|remove|list> [username]
		if len(args) < 3 {
			return "Usage: whitelist <server> <add|remove|list> [user]", nil
		}
		srvName, action := args[1], args[2]
		srvCfg, ok := cfg.Servers[srvName]
		if !ok {
			return fmt.Sprintf("Error: Server '%s' not found in configuration", srvName), nil
		}

		mcCommand := ""
		if action == "list" {
			mcCommand = "whitelist list"
		} else if len(args) >= 4 {
			username := args[3]
			if !ValidateMinecraftUsername(username) {
				return fmt.Sprintf("Error: Invalid Minecraft username: %s", username), nil
			}
			if action == "add" {
				// Java版ユーザー（ドットで始まらない）の場合のみバリデーションとして UUID 解決を試みる
				if !strings.HasPrefix(username, ".") {
					if _, err := ResolveUUID(username); err != nil {
						return fmt.Sprintf("Error: Failed to resolve UUID for %s: %v", username, err), nil
					}
				}
			}
			// remove の場合は名前変更等に対応するため UUID 解決をスキップして直接命令を送る
			mcCommand = fmt.Sprintf("whitelist %s "%s"", action, username)
			log.Printf("[RCON] Sending command: %s", mcCommand)
		} else {
			return "Error: Username required for add/remove", nil
		}

		client, err := DialRCON(srvCfg.Network, srvCfg.Address, srvCfg.Password)
		if err != nil {
			return "", err
		}
		defer client.Close()
		return client.Execute(mcCommand)

	case "status":
		return fmt.Sprintf("Bridge is running. Registered Servers: %s", strings.Join(getServerNames(cfg), ", ")), nil

	default:
		return fmt.Sprintf("Unknown command: %s", command), nil
	}
}

func StartSocketServer(path string, db *Database, cfg *Config) {
	if err := os.RemoveAll(path); err != nil {
		log.Fatal(err)
	}

	l, err := net.Listen("unix", path)
	if err != nil {
		log.Fatal("Socket listen error:", err)
	}
	defer l.Close()

	if err := os.Chmod(path, 0660); err != nil {
		log.Printf("Warning: Failed to chmod socket: %v", err)
	}

	log.Printf("[SOCKET] Management socket listening on %s", path)

	for {
		conn, err := l.Accept()
		if err != nil {
			continue
		}
		go handleSocketConnection(conn, db, cfg)
	}
}

func handleSocketConnection(c net.Conn, db *Database, cfg *Config) {
	defer c.Close()
	scanner := bufio.NewScanner(c)
	for scanner.Scan() {
		input := scanner.Text()
		response, err := ProcessCommand(input, db, cfg)
		if err != nil {
			c.Write([]byte(fmt.Sprintf("ERROR: %v\n", err)))
		} else {
			c.Write([]byte(response + "\n"))
		}
	}
}

func getServerNames(cfg *Config) []string {
	names := make([]string, 0, len(cfg.Servers))
	for k := range cfg.Servers {
		names = append(names, k)
	}
	return names
}