package main

import (
	"context"
	"log"
	"os"
	"time"
	_ "time/tzdata"
)

func init() {
	// 1. 環境変数「TZ」から値を取得（設定されていなければデフォルト値を設定）
	tzEnv := os.Getenv("TZ")
	if tzEnv == "" {
		tzEnv = "Asia/Tokyo" // デフォルト値
	}

	// 2. タイムゾーン名からLocationオブジェクトを生成
	loc, err := time.LoadLocation(tzEnv)
	if err != nil {
		// 変換に失敗した場合は、安全のためUTCや固定値を割り当てる
		loc = time.UTC
	}

	// 3. グローバルのローカルタイムゾーンを上書き
	time.Local = loc
}

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	if err := run(context.Background(), os.Args[1:]); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
