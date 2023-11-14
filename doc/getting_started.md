# Getting started

## Prerequisites
* Ubuntu >=20.04
* Golang >= 1.16
* DHCP server must be configured to offer IP addresses to ADC boards
* Reference software (AFI GUI software) must be available (used for configuring ADC boards)

## Build
`go-adc` software is built into a single binary which provides all the available functionality
via the command line interface. To build the `go-adc` binary use the command.
```bash
make build
```
Then copy the binary file `bin/go-adc` to the machine (Ubuntu 20.04 or similar) where it is assumed to run. The machine must have access to the IP network where ADC boards are attached.

## Initial configuration
`go-adc` stores its configuration and state database files in the `~/.go-adc` directory.

To create a default configuration file use the command:
```bash
go-adc config init
```
This command will create the file `~/.go-adc/config` with the minimal default configuration
```yaml
devices: []
discoverIP: 239.192.1.1
discoverIface: eth0
ip: 192.168.1.100
logLevel: info
```

The fields in the file are:
- `devices` is the list of devices identified by the device name and the device IP
- `discoverIP` is the multicast IP where the discovery server listens to discovery messages
- `discoverIface` is the network interface on the machine attached to the IP network with ADC devices
- `ip` is the IP address of the machine where the `go-adc` is run

## Discover boards
To start the discovery server use the command
```bash
go-adc discover start
```
While it is running it receives the discovery messages from the ADC boards and puts the discovery information to the local database. To see the list of ADC boards discovered you can send the http request using the following command
```bash
curl http://<ip>:8003/api/devices
```
Or you can use the `go-adc` command line interface
```bash
go-adc discover list
```
Now when you get the list of boards available you can put their names and IP addresses to the config file `~/.go-adc/config`
```yaml
devices:
  - name: first
    ip: 192.168.1.101
  - name: second
    ip: 192.168.1.102
discoverIP: 239.192.1.1
discoverIface: eth0
ip: 192.168.1.100
logLevel: info
```

## Start control server
To start the control server use the command
```bash
go-adc control start
```
The control server reads and writes control registries on the ADC boards. One of these registries can be used to start and stop ADC streaming. But before starting the streaming you have to start the mstream server

## Start mstream server
To start the mstream server use the command
```
go-adc mstream start
```

## Start and stop streaming
Now you are ready to start the streaming. Use the command
```bash
go-adc control mstream start --dir <data_directory> --file-prefix <run_prefix>
```
where `<data_directory>` is the path to the directory where you would like to store the files with the data and `<run_prefix>` is the desirable prefix for the data files to easily identify the run.

To stop the streaming use the command
```bash
go-adc control mstream stop
```

