package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

var AdminGuildID string

func RegisterCommands(s *discordgo.Session, guildID string) {
	AdminGuildID = guildID

	// グローバルコマンド
	globalCommands := []*discordgo.ApplicationCommand{
		{
			Name:        "join",
			Description: "Join the Minecraft server and get the member role",
		},
		{
			Name:        "bridge-link",
			Description: "Link this Discord server to a Minecraft server",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "token",
					Description: "Invitation token",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionRole,
					Name:        "role",
					Description: "The role required to manage the Minecraft server via this bot",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionChannel,
					Name:        "channel",
					Description: "Restrict commands to this specific channel",
					Required:    false,
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

	// 管理者用コマンド
	adminCommands := []*discordgo.ApplicationCommand{
		{
			Name:        "token",
			Description: "Manage invitation tokens",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "create",
					Description: "Generate a new invitation token",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "server",
							Description: "Minecraft server name",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "name",
							Description: "Name for this invitation",
							Required:    true,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "list",
					Description: "List all active invitation tokens",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "revoke",
					Description: "Revoke an invitation token",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "token",
							Description: "Token to revoke",
							Required:    true,
						},
					},
				},
			},
		},
	}

	s.ApplicationCommandBulkOverwrite(s.State.User.ID, "", globalCommands)
	if guildID != "" {
		s.ApplicationCommandBulkOverwrite(s.State.User.ID, guildID, adminCommands)
	}
}

func AddHandlers(s *discordgo.Session, db *Database, cfg *Config) {
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}

		data := i.ApplicationCommandData()
		switch data.Name {
		case "join":
			handleJoin(s, i, db)
		case "bridge-link":
			handleLink(s, i, db)
		case "whitelist":
			handleWhitelist(s, i, db, cfg)
		case "token":
			handleTokenCommands(s, i, db, cfg)
		}
	})
}

func handleJoin(s *discordgo.Session, i *discordgo.InteractionCreate, db *Database) {
	_, roleID, _, err := db.GetLinkInfo(i.GuildID)
	if err != nil || roleID == "" {
		respondWithError(s, i, "This server is not properly linked or no management role is configured.")
		return
	}

	err = s.GuildMemberRoleAdd(i.GuildID, i.Member.User.ID, roleID)
	if err != nil {
		respondWithError(s, i, fmt.Sprintf("Failed to assign role: %v", err))
		return
	}

	respondWithSuccess(s, i, fmt.Sprintf("✅ Welcome! You have been granted the <@&%s> role.", roleID))
}

func handleLink(s *discordgo.Session, i *discordgo.InteractionCreate, db *Database) {
	options := i.ApplicationCommandData().Options
	token := options[0].StringValue()
	role := options[1].RoleValue(s, i.GuildID)
	
	channelID := ""
	if len(options) > 2 {
		channelID = options[2].ChannelValue(s).ID
	}

	targetServer, err := db.LinkGuild(i.GuildID, token, role.ID, channelID)
	if err != nil {
		respondWithError(s, i, err.Error())
		return
	}

	msg := fmt.Sprintf("✅ Linked to **%s**. Management role: <@&%s>.", targetServer, role.ID)
	if channelID != "" {
		msg += fmt.Sprintf(" Commands are restricted to <#%s>.", channelID)
	}
	respondWithSuccess(s, i, msg)
}

func handleWhitelist(s *discordgo.Session, i *discordgo.InteractionCreate, db *Database, cfg *Config) {
	// 1. サーバーがリンクされているか確認
	targetServer, roleID, allowedChannelID, err := db.GetLinkInfo(i.GuildID)
	if err != nil {
		respondWithError(s, i, "This Discord server is not linked to any Minecraft server. Use `/bridge-link` first.")
		return
	}

	// 2. チャンネル制限のチェック
	if allowedChannelID != "" && i.ChannelID != allowedChannelID {
		respondWithError(s, i, fmt.Sprintf("This command can only be used in <#%s>.", allowedChannelID))
		return
	}

	// 3. 管理ロールを持っているかチェック
	if roleID != "" {
		hasRole := false
		for _, r := range i.Member.Roles {
			if r == roleID {
				hasRole = true
				break
			}
		}
		if !hasRole {
			respondWithError(s, i, fmt.Sprintf("You need the <@&%s> role to use this command. Use `/join` to get it.", roleID))
			return
		}
	}

	// 4. 共通ロジック (socketと同一) を呼び出すための文字列を生成
	sub := i.ApplicationCommandData().Options[0]
	cmdText := ""
	if sub.Name == "list" {
		cmdText = fmt.Sprintf("whitelist %s list", targetServer)
	} else {
		cmdText = fmt.Sprintf("whitelist %s %s %s", targetServer, sub.Name, sub.Options[0].StringValue())
	}

	resp, err := ProcessCommand(cmdText, db, cfg)
	if err != nil {
		respondWithError(s, i, err.Error())
	} else {
		respondWithSuccess(s, i, fmt.Sprintf("🎮 **Minecraft Response:**\n```\n%s\n```", resp))
	}
}

func handleAdminCommands(s *discordgo.Session, i *discordgo.InteractionCreate, db *Database, cfg *Config) {
	if i.GuildID != AdminGuildID {
		respondWithError(s, i, "This command can only be used in the admin server.")
		return
	}

	data := i.ApplicationCommandData()
	cmdText := data.Name
	for _, opt := range data.Options {
		cmdText += " " + opt.StringValue()
	}

	resp, err := ProcessCommand(cmdText, db, cfg)
	if err != nil {
		respondWithError(s, i, err.Error())
	} else {
		respondWithSuccess(s, i, resp)
	}
}

func handleTokenCommands(s *discordgo.Session, i *discordgo.InteractionCreate, db *Database, cfg *Config) {
	if i.GuildID != AdminGuildID {
		respondWithError(s, i, "This command can only be used in the admin server.")
		return
	}

	sub := i.ApplicationCommandData().Options[0]
	cmdText := ""
	switch sub.Name {
	case "create":
		cmdText = fmt.Sprintf("invite-create %s %s", sub.Options[0].StringValue(), sub.Options[1].StringValue())
	case "list":
		cmdText = "invite-list"
	case "revoke":
		cmdText = fmt.Sprintf("invite-revoke %s", sub.Options[0].StringValue())
	}

	resp, err := ProcessCommand(cmdText, db, cfg)
	if err != nil {
		respondWithError(s, i, err.Error())
	} else {
		respondWithSuccess(s, i, resp)
	}
}

func respondWithError(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("❌ **Error:** %s", msg),
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func respondWithSuccess(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
		},
	})
}
