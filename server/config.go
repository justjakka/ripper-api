package server

type ServerConfig struct {
	BindAddr    string
	WebDir      string
	BindWrapper string
	KeyList     *[]string
}
