package server

import (
	"net/http"

	"micro-front/internal/config"
)

// Server は HTTP ルーティングと実行設定をまとめたサーバ本体です。
type Server struct {
	cfg config.Config
	mux *http.ServeMux
}
