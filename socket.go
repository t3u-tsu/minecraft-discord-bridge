package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

type WhitelistEntry struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

// コマンド実行の核となる関数
func ProcessCommand(input string, db *Database, cfg *Config) (string, error) {
	args := strings.Fields(input)
	if len(args) == 0 {
		return "", nil
	}

	command := args[0]
	log.Printf("[CORE] Processing command: %s", input)

	switch command {
	case "invite-create":
		if len(args) < 3 {
			return "Usage: invite-create <server_name> <invitation_name>", nil
		}
		srvName := args[1]
		if _, ok := cfg.Servers[srvName]; !ok {
			return fmt.Sprintf("Error: Server '%s' not found", srvName), nil
		}
		inviteName := strings.Join(args[2:], " ")
		token := GenerateToken(srvName)
		if err := db.CreateInvitation(token, srvName, inviteName); err != nil {
			return "", err
		}
		return fmt.Sprintf("Token Created: %s", token), nil

	case "invite-list":
		list, err := db.ListInvitations()
		if err != nil {
			return "", err
		}
		var sb strings.Builder
		for _, i := range list {
			sb.WriteString(fmt.Sprintf("- %s | %s | %s\n", i.Token, i.TargetServer, i.Name))
		}
		return sb.String(), nil

	case "invite-revoke":
		if len(args) < 2 {
			return "Usage: invite-revoke <token>", nil
		}
		if err := db.RevokeInvitation(args[1]); err != nil {
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
		var sb strings.Builder
		for _, l := range list {
			sb.WriteString(fmt.Sprintf("- Guild: %s | Server: %s | Role: %s | Channel: %s\n", l.GuildID, l.TargetServer, l.RoleID, l.ChannelID))
		}
		return sb.String(), nil

	case "whitelist":
		if len(args) < 3 {
			return "Usage: whitelist <server> <add|remove|list> [user]", nil
		}
		srvName, action := args[1], args[2]
		srvCfg, ok := cfg.Servers[srvName]
		if !ok {
			return fmt.Sprintf("Error: Server '%s' not found", srvName), nil
		}

		if action == "list" {
			client, err := DialRCON(srvCfg.Network, srvCfg.Address, srvCfg.Password)
			if err != nil {
				log.Printf("[CORE] RCON Connection failed for %s: %v", srvName, err)
				return "", err
			}
			defer client.Close()
			return client.Execute("whitelist list")
		}

		if len(args) < 4 {
			return "Error: Username required", nil
		}
		username := args[3]

		if action == "add" {
			if !ValidateMinecraftUsername(username) {
				return "Error: Invalid username", nil
			}
			if !strings.HasPrefix(username, ".") {
				if _, err := ResolveUUID(username); err != nil {
					log.Printf("[CORE] UUID resolution failed for %s: %v", username, err)
					return fmt.Sprintf("Error: UUID resolution failed: %v", err), nil
				}
			}
			client, err := DialRCON(srvCfg.Network, srvCfg.Address, srvCfg.Password)
			if err != nil {
				log.Printf("[CORE] RCON Connection failed for %s: %v", srvName, err)
				return "", err
			}
			defer client.Close()
			return client.Execute(fmt.Sprintf("whitelist add %s", username))
		}

		if action == "remove" {
			if srvCfg.WhitelistPath == "" {
				return "Error: whitelist_path not configured", nil
			}
			data, err := os.ReadFile(srvCfg.WhitelistPath)
			if err != nil {
				log.Printf("[CORE] Failed to read whitelist file: %v", err)
				return fmt.Sprintf("Error reading whitelist: %v", err), nil
			}
			var list []WhitelistEntry
			if err := json.Unmarshal(data, &list); err != nil {
				log.Printf("[CORE] Failed to parse whitelist JSON: %v", err)
				return fmt.Sprintf("Error parsing whitelist: %v", err), nil
			}

			newList := []WhitelistEntry{}
			found := false
			for _, e := range list {
				if strings.EqualFold(e.Name, username) {
					found = true
					continue
				}
				newList = append(newList, e)
			}
			if !found {
				return "Error: Player not in whitelist file", nil
			}

			newData, _ := json.MarshalIndent(newList, "", "  ")
			if err := os.WriteFile(srvCfg.WhitelistPath, newData, 0644); err != nil {
				log.Printf("[CORE] Failed to write whitelist file: %v", err)
				return fmt.Sprintf("Error writing whitelist: %v", err), nil
			}

			client, err := DialRCON(srvCfg.Network, srvCfg.Address, srvCfg.Password)
			if err != nil {
				log.Printf("[CORE] RCON Connection failed for %s: %v", srvName, err)
				return "", err
			}
			defer client.Close()
			return client.Execute("whitelist reload")
		}

	case "status":
		return fmt.Sprintf("Bridge is running. Registered Servers: %s", strings.Join(getServerNames(cfg), ", ")), nil
	}
	return "Unknown command", nil
}

func StartSocketServer(path string, db *Database, cfg *Config) {
	os.RemoveAll(path)
	l, err := net.Listen("unix", path)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	os.Chmod(path, 0660)
	log.Printf("[SOCKET] Listening on %s", path)
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
		res, err := ProcessCommand(scanner.Text(), db, cfg)
		if err != nil {
			c.Write([]byte("Error: " + err.Error() + "\n"))
		} else {
			c.Write([]byte(res + "\n"))
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