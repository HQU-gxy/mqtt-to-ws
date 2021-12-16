# MQTT to Websocket Bridge with MongoDB

## Structure

```txt
├── controller          # gin router controller
│   └── controller.go
├── docs                # swagger documention generated by `swag init`
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
├── logger              # zap logger
│   └── logger.go
├── main.go
├── makefile
├── model               # mongoDB interface
│   └── model.go
└── utils               # utils for websocket
    ├── client.go
    └── hub.go
```

## Build

Refer to [makefile](makefile)

This project use [swaggo/swag](http://github.com/swaggo/swag) to generate swagger documention. You can install it by:

```bash
go get -u github.com/swaggo/swag/cmd/swag

# 1.16 or newer
go install github.com/swaggo/swag/cmd/swag@latest
```

If you modified any swagger comments, you need to run `swag init` before building.

## Usage

make sure you have mongoDB installed and running.

```txt
Usage: D:\Dev\Desktop\golang-ws\bin.exe [-a addr:port] [-A addr:port] [-D database] [-M url] [-s addr:port] [-w path] [parameters ...]
 -a, --addr-http=addr:port
       HTTP API address (default: :8080)
 -A, --addr-mqtt=addr:port
       MQTT broker address (default: :1883)
 -D, --database=database
       Database name (default: mqtt)
 -M, --mongo-url=url
       MongoDB connection URL (default: mongodb://localhost:27017)
       mongodb://[username:password@]host1[:port1][,...hostN[:portN]][/[defaultauthdb][?options]]
 -s, --addr-swagger=addr:port
       Swagger BaseURL -- change this if swagger is not working
       correctly
 -w, --websocket=path
       Websocket listening path -- default '/ws'
```

## API documentation

### HTTP

check `docs/swagger.yaml` for HTTP API documentation.
Or use access the swagger UI by [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)

### Websocket

The default websocket url is `ws://localhost:8080/ws`.

Any MQTT package will be automatically forwarded to the websocket as follows:

```json
{
    "topic": "temperature",
    "payload": "23.5"
}
```

## Todo

- [ ] Record Client ID
- [x] makefile
- [ ] config file
- [x] swagger documention
- [x] refactor (split main into multiple files)
- [ ] unit tests
