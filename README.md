# Minecraft Discord Bridge

A versatile Minecraft-Discord integration tool written in Go, optimized for NixOS environments.
It features a multi-tenant management system that allows secure delegation of Minecraft server administration to specific Discord guilds.

## Key Features

- **Multi-tenant Management**: Securely link Discord guilds to specific Minecraft servers using invitation tokens.
- **Whitelist Operations**: Manage Minecraft whitelists via Discord slash commands (`/whitelist add|remove|list`).
- **UUID Resolution**: Automatic validation of player names and UUIDs via Mojang API.
- **Local Management Socket**: Control RCON and issue tokens directly from the server's command line.
- **Native RCON Support**: Works over both TCP and Unix Domain Sockets.

---

## Tutorial: Getting Started

### 1. Create a Discord Bot
1. Go to the [Discord Developer Portal](https://discord.com/developers/applications).
2. Create a **New Application** and select **Bot** from the left menu.
3. Click **Reset Token** to get your bot token (save this for your config).
4. In the **Privileged Gateway Intents** section, enable `Message Content Intent`.

### 2. Invite the Bot
1. Go to **OAuth2 -> URL Generator**.
2. Select `bot` and `applications.commands` scopes.
3. Select `Send Messages` and `Read Message History` permissions.
4. Open the generated URL in your browser and invite the bot to your server.

### 3. Build and Configure
```bash
# Build the binary
go build -o bridge .

# Create configuration
cp config.toml.sample config.toml
# Edit config.toml with your tokens and RCON details
```

### 4. Link Your Server
1. Start the bridge: `./bridge -c config.toml`
2. Issue an invitation token on your management server:
   `echo "invite-create <server_name>" | nc -U /run/bridge.sock`
3. Run the link command in Discord:
   `/bridge-link token:<your_token_here>`

---

## Configuration

### config.toml
Defines RCON connection details for each Minecraft server.

### Environment Variables
Sensitive values can be overridden via environment variables (recommended for NixOS/Docker).
- `DISCORD_TOKEN`: Your bot token
- `DISCORD_ADMIN_GUILD_ID`: Discord Guild ID for admin commands
- `RCON_PASS_<server_name>`: RCON password for each specific server

---

## Command Reference

### Discord Slash Commands
- `/invite-create`: [Admin Only] Generate a new invitation token.
- `/bridge-link`: Pair a Discord guild with a Minecraft server.
- `/whitelist add|remove|list`: Manage the server whitelist.

### Unix Socket Commands
- `invite-create <server>`: Issue a token locally.
- `status`: Check bridge and server connection status.
- `whitelist <server> <add|remove|list> [user]`: Direct RCON control.

## License

0BSD (BSD Zero Clause License) - See `LICENSE` for details.
