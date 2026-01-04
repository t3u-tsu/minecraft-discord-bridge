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
			Name:        "invite-create",
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
			Name:        "invite-list",
			Description: "List all active invitation tokens",
		},
		{
			Name:        "invite-revoke",
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
		case "invite-create", "invite-list", "invite-revoke":
			handleAdminCommands(s, i, db, cfg)
		}
	})
}

// 参加表明コマンド: 管理ロールを付与する
func handleJoin(s *discordgo.Session, i *discordgo.InteractionCreate, db *Database) {
	_, roleID, err := db.GetLinkInfo(i.GuildID)
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

	targetServer, err := db.LinkGuild(i.GuildID, token, role.ID)
	if err != nil {
		respondWithError(s, i, err.Error())
		return
	}

	respondWithSuccess(s, i, fmt.Sprintf("✅ Linked to **%s**. Management role set to <@&%s>.", targetServer, role.ID))
}

func handleWhitelist(s *discordgo.Session, i *discordgo.InteractionCreate, db *Database, cfg *Config) {
	// 管理ロールを持っているかチェック
	_, roleID, err := db.GetLinkInfo(i.GuildID)
	if err == nil && roleID != "" {
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

	// 共通ロジック (socketと同一) を呼び出すための文字列を生成
	sub := i.ApplicationCommandData().Options[0]
	targetServer, _, _ := db.GetLinkInfo(i.GuildID)
	
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
