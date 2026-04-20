package config

// Config はアプリケーション起動時の設定値を保持します。
type Config struct {
	Addr            string
	AdminStaticDir  string
	PublicStaticDir string
	DataDir         string
	DBPath          string
}
