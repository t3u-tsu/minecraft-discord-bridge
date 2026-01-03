package main

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

var (
	AdminGuildID string // 管理用コマンドを許可する Discord サーバー ID
)

func RegisterCommands(s *discordgo.Session, guildID string) {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "bridge-link",
			Description: "Link this Discord server to a Minecraft server using an invitation token",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "token",
					Description: "Invitation token",
					Required:    true,
				},
			},
		},
		{
			Name:        "whitelist",
			Description: "Manage Minecraft whitelist",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "add",
					Description: "Add a player to the whitelist",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "username",
							Description: "Minecraft username",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "remove",
					Description: "Remove a player from the whitelist",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "username",
							Description: "Minecraft username",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "list",
					Description: "List all whitelisted players",
				},
			},
		},
	}

	adminCommands := []*discordgo.ApplicationCommand{
		{
			Name:        "invite-create",
			Description: "Generate a new invitation token for a Minecraft server",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "server",
					Description: "Minecraft server name (e.g., nitac23s, lobby)",
					Required:    true,
				},
			},
		},
	}

	for _, v := range commands {
		_, err := s.ApplicationCommandCreate(s.State.User.ID, "", v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		}
	}

	if AdminGuildID != "" {
		for _, v := range adminCommands {
			_, err := s.ApplicationCommandCreate(s.State.User.ID, AdminGuildID, v)
			if err != nil {
				log.Printf("Cannot create admin command '%v': %v", v.Name, err)
			}
		}
	}
}

func AddHandlers(s *discordgo.Session, db *Database, cfg *Config) {
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			data := i.ApplicationCommandData()
			switch data.Name {
			case "bridge-link":
				handleLink(s, i, db)
			case "whitelist":
				handleWhitelist(s, i, db, cfg)
			case "invite-create":
				handleInviteCreate(s, i, db)
			}
		}
	})
}

func handleLink(s *discordgo.Session, i *discordgo.InteractionCreate, db *Database) {
	token := i.ApplicationCommandData().Options[0].StringValue()
	targetServer, err := db.LinkGuild(i.GuildID, token)

	msg := ""
	if err != nil {
		msg = fmt.Sprintf("❌ Error: %v", err)
	} else {
		msg = fmt.Sprintf("✅ Success! This Discord server is now linked to Minecraft server: **%s**", targetServer)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral, // 本人のみに表示
		},
	})
}

func handleWhitelist(s *discordgo.Session, i *discordgo.InteractionCreate, db *Database, cfg *Config) {
	// 1. サーバーがリンクされているか確認
	targetServer, err := db.GetTargetServer(i.GuildID)
	if err != nil {
		respondWithError(s, i, err.Error())
		return
	}

	// 2. サーバーの RCON 設定を取得
	srvCfg, ok := cfg.Servers[targetServer]
	if !ok {
		respondWithError(s, i, fmt.Sprintf("Configuration for server '%s' not found", targetServer))
		return
	}

	// 3. コマンドの解析
	subCommand := i.ApplicationCommandData().Options[0]
	action := subCommand.Name // "add", "remove", or "list"

	mcCommand := ""
	if action == "list" {
		mcCommand = "whitelist list"
	} else {
		username := subCommand.Options[0].StringValue()
		// UUID の解決 (add/remove のみ)
		uuid, err := ResolveUUID(username)
		if err != nil {
			respondWithError(s, i, fmt.Sprintf("Failed to resolve UUID for %s: %v", username, err))
			return
		}
		log.Printf("Resolved UUID for %s: %s", username, uuid)
		mcCommand = fmt.Sprintf("whitelist %s %s", action, username)
	}

	// 4. RCON 実行
	client, err := DialRCON(srvCfg.Network, srvCfg.Address, srvCfg.Password)
	if err != nil {
		respondWithError(s, i, fmt.Sprintf("Failed to connect to Minecraft server: %v", err))
		return
	}
	defer client.Close()

	response, err := client.Execute(mcCommand)
	if err != nil {
		respondWithError(s, i, fmt.Sprintf("Failed to execute command: %v", err))
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("🎮 **Minecraft Response:**\n```\n%s\n```", response),
		},
	})
}

func respondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("❌ Error: %s", msg),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func handleInviteCreate(s *discordgo.Session, i *discordgo.InteractionCreate, db *Database) {
	// 権限チェックは RegisterCommands 側の Guild 制限で行っているが
	// 追加でユーザーIDチェックなども入れるとより安全
	serverName := i.ApplicationCommandData().Options[0].StringValue()
	token := "MC-" + serverName + "-" + fmt.Sprintf("%d", i.ID)[:6] // 簡易的なトークン生成

	err := db.CreateInvitation(token, serverName)
	msg := ""
	if err != nil {
		msg = fmt.Sprintf("❌ Failed to create invitation: %v", err)
	} else {
		msg = fmt.Sprintf("🎫 New invitation token for **%s**:\n`%s`", serverName, token)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
		},
	})
}