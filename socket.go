package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

func StartSocketServer(path string, db *Database, cfg *Config) {
	// 既存のソケットファイルがあれば削除
	if err := os.RemoveAll(path); err != nil {
		log.Fatal(err)
	}

	l, err := net.Listen("unix", path)
	if err != nil {
		log.Fatal("listen error:", err)
	}
	defer l.Close()

	// パーミッションを適切に設定（t3uユーザーや特定のグループが叩けるように）
	if err := os.Chmod(path, 0660); err != nil {
		log.Printf("Warning: Failed to chmod socket: %v", err)
	}

	log.Printf("Management socket listening on %s", path)

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Print("accept error:", err)
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
		log.Printf("Received from socket: %s", input)

		args := strings.Fields(input)
		if len(args) == 0 {
			continue
		}

		response := ""
		switch args[0] {
		case "invite-create":
			if len(args) < 2 {
				response = "Usage: invite-create <server_name>\n"
			} else {
				token := GenerateToken(args[1])
				err := db.CreateInvitation(token, args[1])
				if err != nil {
					response = "Error: " + err.Error() + "\n"
				} else {
					response = "Token created: " + token + "\n"
				}
			}
		case "whitelist":
			// whitelist <server_name> <add|remove|list> [username]
			if len(args) < 3 {
				response = "Usage: whitelist <server_name> <add|remove|list> [username]\n"
			} else {
				srvName := args[1]
				action := args[2]
				srvCfg, ok := cfg.Servers[srvName]
				if !ok {
					response = "Error: Server configuration not found\n"
				} else {
					mcCommand := ""
					if action == "list" {
						mcCommand = "whitelist list"
					} else if len(args) >= 4 {
						mcCommand = fmt.Sprintf("whitelist %s %s", action, args[3])
					} else {
						response = "Error: Username required for add/remove\n"
					}

					if mcCommand != "" {
						client, err := DialRCON(srvCfg.Network, srvCfg.Address, srvCfg.Password)
						if err != nil {
							response = "Error: " + err.Error() + "\n"
						} else {
							res, err := client.Execute(mcCommand)
							client.Close()
							if err != nil {
								response = "Error: " + err.Error() + "\n"
							} else {
								response = res + "\n"
							}
						}
					}
				}
			}
		case "status":
			response = "Bridge is running. Registered servers: " + strings.Join(getServerNames(cfg), ", ") + "\n"
		default:
			response = "Unknown command via socket\n"
		}

		c.Write([]byte(response))
	}
}

func getServerNames(cfg *Config) []string {
	names := make([]string, 0, len(cfg.Servers))
	for k := range cfg.Servers {
		names = append(names, k)
	}
	return names
}
