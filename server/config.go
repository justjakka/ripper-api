package server

type ServerConfig struct {
	BindAddr    string
	WebDir      string
	BindWrapper string
	KeyList     *[]string
}

type Job struct {
	status  int
	message string
}
