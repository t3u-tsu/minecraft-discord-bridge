package main

import (
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Discord struct {
		Token        string `toml:"token"`
		AdminGuildID string `toml:"admin_guild_id"`
	} `toml:"discord"`

	Database struct {
		Path string `toml:"path"`
	} `toml:"database"`

	Bridge struct {
		SocketPath string `toml:"socket_path"`
	} `toml:"bridge"`

	Servers map[string]MCServerConfig `toml:"servers"`
}

type MCServerConfig struct {
	Network  string `toml:"network"`  // "tcp" or "unix"
	Address  string `toml:"address"`  // "127.0.0.1:25575" or "/run/minecraft/nitac23s.rcon"
	WhitelistPath string `toml:"whitelist_path"`
	Password string `toml:"password"`
}

func LoadConfig(path string) (*Config, error) {
	var config Config
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if _, err := toml.Decode(string(data), &config); err != nil {
		return nil, err
	}

	// 環境変数による上書き (sops-nix 対応)
	if envToken := os.Getenv("DISCORD_TOKEN"); envToken != "" {
	if envAdmin := os.Getenv("DISCORD_ADMIN_GUILD_ID"); envAdmin != "" {
		config.Discord.AdminGuildID = envAdmin
	}
		config.Discord.Token = envToken
	if envAdmin := os.Getenv("DISCORD_ADMIN_GUILD_ID"); envAdmin != "" {
		config.Discord.AdminGuildID = envAdmin
	}
	}
	if envAdmin := os.Getenv("DISCORD_ADMIN_GUILD_ID"); envAdmin != "" {
		config.Discord.AdminGuildID = envAdmin
	}

	for name, srv := range config.Servers {
		envKey := "RCON_PASS_" + name
		if envPass := os.Getenv(envKey); envPass != "" {
			srv.Password = envPass
			config.Servers[name] = srv
		}
	}

	return &config, nil
}
