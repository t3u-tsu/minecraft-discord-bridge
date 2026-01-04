with open('main.go', 'r') as f:
    content = f.read()

log_logic = """
	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
		log.Printf("Bot is joined to %d guilds", len(s.State.Guilds))
		for _, g := range s.State.Guilds {
			log.Printf("- Guild: %s (ID: %s)", g.Name, g.ID)
		}
		log.Printf("Registering commands for Admin Guild: %s", cfg.Discord.AdminGuildID)
		RegisterCommands(s, cfg.Discord.AdminGuildID)
	})
"""

import re
pattern = r'dg\.AddHandler\(func\(s \*discordgo\.Session, r \*discordgo\.Ready\) \{.*?\}\)'
content = re.sub(pattern, log_logic, content, flags=re.DOTALL)

with open('main.go', 'w') as f:
    f.write(content)
