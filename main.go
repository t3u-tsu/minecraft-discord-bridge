package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
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

	var once sync.Once

	// 接続情報の表示用ハンドラ
	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("[DISCORD] Ready! Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
		
		// Ready イベントで一度だけ確実にコマンド登録を行う
		once.Do(func() {
			log.Println("[DISCORD] Performing initial command registration...")
			RegisterCommands(s, cfg.Discord.AdminGuildID)
			log.Println("[DISCORD] Initial command registration complete.")
		})
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
		log.Println("[DISCORD] Bot connected to gateway.")
	}

	log.Println("Bot is now running. Press CTRL-C to exit.")

	// シグナル待機
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
