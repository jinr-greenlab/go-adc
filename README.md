# Config

```yaml
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
1) mkdir swaggerui
2) Install go-swagger. For detailes how to install go-swagger binary on different platforms proceed next link
https://goswagger.io/install.html
3) Copy content of https://github.com/swagger-api/swagger-ui/tree/master/dist directory into swaggerui dir
4) Put swagger annotation tags into source file describing your APIs and put next string to routes
s.Router.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", http.FileServer(http.Dir("./swaggerui/")))) 
5) Once your start your server API docs will be available on 
http://localhost:your_port/swagger/


