package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var (
	ConfigPath string
)

func init() {
	flag.StringVar(&ConfigPath, "c", "./config.toml", "Configuration file path")
	flag.Parse()
}

func main() {
	// 設定の読み込み
	cfg, err := LoadConfig(ConfigPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// データベースの初期化
	db, err := InitDB(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// 管理用ソケットサーバーをバックグラウンドで開始
	go StartSocketServer(cfg.Bridge.SocketPath, db, cfg)

	// Discord セッションの作成
	dg, err := discordgo.New("Bot " + cfg.Discord.Token)
	if err != nil {
		log.Fatalf("Failed to create Discord session: %v", err)
	}
	dg.Identify.Intents = discordgo.IntentsGuilds | discordgo.IntentsGuildMembers

	// 接続情報の表示用ハンドラ
	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("[DISCORD] Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
		log.Printf("[DISCORD] Bot is joined to %d guilds", len(s.State.Guilds))
	})

	// インタラクションハンドラの追加
	AddHandlers(dg, db, cfg)

	// 接続開始
	err = dg.Open()
	if err != nil {
		log.Printf("[DISCORD] Failed to open connection: %v", err)
		log.Println("Continuing in local management mode...")
	} else {
		defer dg.Close()
		log.Println("[DISCORD] Bot connected. Registering commands...")
		
		// コマンドの登録を Open の直後に行う (Ready を待たずに開始)
		RegisterCommands(dg, cfg.Discord.AdminGuildID)
		
		log.Println("[DISCORD] Command registration process initiated.")
	}

	log.Println("Bot is now running. Press CTRL-C to exit.")

	// シグナル待機
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
