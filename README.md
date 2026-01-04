# Minecraft Discord Bridge

A versatile Minecraft-Discord integration tool written in Go, optimized for NixOS. It features a multi-tenant management system that allows secure delegation of Minecraft server administration to specific Discord guilds.

## Key Features

- **Multi-tenant Management**: Securely link Discord guilds to specific Minecraft servers using invitation tokens.
- **Persistent Named Invitations**: Track and reuse tokens by name. Tokens remain valid until manually revoked.
- **Role-based Access Control**: Require specific Discord roles for administration commands.
- **Channel Restriction**: Limit bot commands to a specific Discord channel for each linked server.
- **Onboarding System**: Simple `/join` command to grant the required administration role to users.
- **Link Management**: List all connected Discord guilds and unlink them remotely if necessary.
- **Whitelist Operations**: Manage whitelists via Discord slash commands.
- **Reliable Removal**: Direct `whitelist.json` editing ensures that name-changed or Bedrock players can be removed without issues.
- **UUID Resolution**: Automatic validation of player names via Mojang API.
- **Native RCON**: Support for RCON over both TCP and Unix Domain Sockets.
- **Local Management Socket**: Control everything from the server's command line.
- **Auto-Migration**: Database schema updates are automatically applied on startup.

---

## Getting Started

### 1. Create a Discord Bot
1. Go to the [Discord Developer Portal](https://discord.com/developers/applications).
2. Create an application and navigate to the **Bot** tab.
3. **Important**: Enable **Server Members Intent** under Privileged Gateway Intents.

### 2. Invite the Bot
1. Use the **OAuth2 URL Generator**.
2. Select `bot` and `applications.commands` scopes.
3. Grant `Manage Roles` and `Send Messages` permissions.

### 3. Setup and Pair
1. Configure `config.toml` and start the binary.
2. Issue a token on your admin server: `/token create server:nitac23s name:AdminTeamA`
3. Link a target server: `/bridge-link token:<TOKEN> role:@AdminRole [channel:#channel]`
4. **Important**: In your Discord server settings, ensure the bot's own role (e.g., `minecraft-bridge`) is positioned **above** the management role.

---

## Commands

### Discord Slash Commands
- `/help`: Display usage instructions.
- `/token create <srv> <name>`: [Admin Only] Issue a named token.
- `/token list`: [Admin Only] List active tokens.
- `/token revoke <token>`: [Admin Only] Revoke a token.
- `/links list`: [Admin Only] List all linked Discord guilds.
- `/links unlink <guild_id>`: [Admin Only] Terminate a link with a guild.
- `/bridge-link <token> <role> [channel]`: Link a guild and set its management role/channel.
- `/join`: Request the management role.
- `/whitelist <add|remove|list>`: Manage server whitelist.

### Unix Socket Commands
- `token create`, `token list`, `token revoke`
- `links`, `unlink`
- `status`, `whitelist`

## License

0BSD (BSD Zero Clause License)
