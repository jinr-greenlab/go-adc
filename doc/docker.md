# Getting started using docker

## Prerequisites
* Docker and docker-compose must be installed on the host where you run `go-adc`
* DHCP server must be configured to serve ADC boards
* ADC64 reference software (GUI) must be available (used for configuring boards)

`go-adc` stores its config and state database files in `~/.go-adc`. So when you run `go-adc` inside
docker container you have to mount a host directory to this mount point which is `/root/.go-adc` because
the default user inside the docker image is root.

When you run mstream server you can define an absolute path or a relative path where to save data
received from boards. If you define a relative path the data will be saved to this path relative to
the current working directory which is `/data` for the `go-adc` docker image. So you also have to
mount a host directory to the `/data` mount point.

Let current directory be a directory where you want everything to be put (including config file, state files, data files).

## Initial config file
First you have to create a default empty config file
```
docker run -it --rm --network host -v ./tmp:/root/.go-adc quay.io/kozhukalov/go-adc:adc64 go-adc config init
```

This command will create a default minimal config file `./config`. The owner of this file will be root since
the process inside the docker container is run as root.
```yaml
devices: []
discoverIP: 239.192.1.1
discoverIface: eth0
ip: 192.168.1.100
logLevel: info
```

In this file you have to specify `discoverIface` and `ip` fields which are the name and the IP address of the interface connected to the network where all ADC boards are connected.

*More advanced networking configurations are possible but this
is out of scope of this document. Since the discovery protocol works over UDP and leverages multicast IP
address ADC boards are not necessarily assumed to be conneced to the same L2 segment.*

Note that the device list is empty so far.

## Discover boards
Now you can start the discover server
```
docker run -it --rm --network host -v .:/root/.go-adc  quay.io/kozhukalov/go-adc:adc64 go-adc discover start
```
While it is running you can send a http request to see which boards are discovered
```
curl http://<ip>:8003/api/devices
```
Or you can use `go-adc discover list` command
```
docker run -it --rm --network host -v .:/root/.go-adc  quay.io/kozhukalov/go-adc:adc64 go-adc discover list
```

Now using the discover data you can finalize the configuration
```yaml
devices:
- name: device1 # you can use device serial number its name
  ip: 192.168.1.151
- name: device2
  ip: 192.168.1.197
discoverIP: 239.192.1.1
discoverIface: eth0
ip: 192.168.1.100
logLevel: info
```

## All three servers
Now once all necessary configuration fields are specified everything is ready to get started.

Since we need to run at least two independent components (control and mstream servers) it is convenient to use docker-compose. Use the following docker-compose file as an example
```yaml
version: '3.3'
services:
  mstream-server: &mstream-server
    environment:
    - TZ=Europe/Moscow
    image: quay.io/kozhukalov/go-adc:latest
    restart: "no"
    volumes:
    - .:/data
    - .:/root/.go-adc
    network_mode: host
    command: go-adc mstream start
  control-server:
    <<: *mstream-server
    command: go-adc control start
  discover-server:
    <<: *mstream-server
    command: go-adc discover start
```

Just save the above content to `./docker-compose.yaml` file and run the following command to run in foreground
```
docker-compose up
```
or this command to run all three servers in background
```
docker-compose up -d
```

## Data aquisition
While servers are running you can use these commands to start and stop a data aquisition session.

To start data aquisition session
```
docker run -it --rm --network host -v .:/root/.go-adc  quay.io/kozhukalov/go-adc:adc64 go-adc control mstream start --dir some/relative/path --file-prefix funny_run
```

For stop data aquisition session
```
docker run -it --rm --network host -v .:/root/.go-adc  quay.io/kozhukalov/go-adc:adc64 go-adc control mstream stop
```
