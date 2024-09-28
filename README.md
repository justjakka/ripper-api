# Ripper-api

web api for ALAC downloader. supports both frida client and wrapper

original ripper code from https://github.com/alacleaker/apple-music-alac-downloader

### Usage:
```
NAME:
   ripper-api serve - Run the HTTP/asynq server

USAGE:
   ripper-api serve [command options]

OPTIONS:
   --port value, -p value                                     Port to bind the HTTP listener to (default: 8080) [$PORT]
   --web-dir value, -d value                                  Temporary directory for content serving [$WEB_DIR]
   --wrappers value, -w value [ --wrappers value, -w value ]  Wrapper addresses and ports [$WRAPPERS]
   --key-db value, -k value                                   File with valid api keys [$KEY_DB]
   --redis value, -r value                                    Address and port of redis [$REDIS_ADDRESS]
   --redis-pw value, --pw value                               Redis DB password [$REDIS_PASSWORD]
   --config value, -c value                                   Path for config file [$RIPPER_CONFIG]
```

### Config file example (in TOML format):
```
Port = 8100
Redis = "127.0.0.1:6379"
Wrappers = [ "127.0.0.1:10200" ]
Webdir = "/web"
RedisPw = "123"
Keyfile = "/keys"
```
