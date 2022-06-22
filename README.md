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
