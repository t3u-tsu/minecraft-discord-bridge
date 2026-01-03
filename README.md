# Minecraft Discord Bridge

A versatile Minecraft-Discord integration tool written in Go, optimized for NixOS environments. It features a multi-tenant management system that allows secure delegation of Minecraft server administration to specific Discord guilds.

## Features

- **Multi-tenant Management**: Dynamically link Discord guilds to specific Minecraft servers using invitation tokens.
- **Whitelist Operations**: Manage whitelists via Discord slash commands (`/whitelist <add|remove|list>`).
- **UUID Resolution**: Automatic validation of player names and UUIDs via Mojang API.
- **Dual Interfaces**: Support for Discord slash commands and a local Unix domain socket for management.
- **Native RCON**: Support for RCON over both TCP and Unix Domain Sockets.

## Architecture

1.  **SQLite DB**: Stores dynamic links between Discord guilds and Minecraft servers.
2.  **TOML Configuration**: Static settings for Discord tokens and Minecraft RCON credentials.
3.  **Unix Socket**: A local management interface accessible via tools like `nc`.

## Usage

### 1. Build
```bash
nix develop
go build -o bridge .
```

### 2. Configuration
Create a `config.toml` with your Discord token and server RCON details.

### 3. Run
```bash
./bridge -c config.toml
```

## License

0BSD (BSD Zero Clause License)
