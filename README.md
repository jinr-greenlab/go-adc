# Config

```yaml
logLevel: debug # must be one of error, warning, info, debug
devices:
- name: one
  ip: 192.168.1.101
- name: two
  ip: 192.168.1.102
discoverIP: 239.192.1.1
discoverIface: eth0
ip: 192.168.1.100
```

# Install tools

## Golint
```sh
go get -u golang.org/x/lint/golint
```

## Golangci-lint
```sh
go get -u github.com/golangci/golangci-lint/cmd/golangci-lint@v1.46.2
```

## go-swagger
1. Install go-swagger. For detailes how to install go-swagger binary on different platforms proceed to the next link
https://goswagger.io/install.html
1. Put swagger annotation tags into source file describing your APIs
1. Generate swagger specificatioin file by typing
```sh
make swagger
```
1. When your start your server API docs will be available on
http://localhost:your_port/swagger/