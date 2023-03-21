ARG BUILD_IMAGE=golang:1.19.7-bullseye
ARG RELEASE_IMAGE=debian:bullseye

FROM ${BUILD_IMAGE} as builder

WORKDIR /src
SHELL [ "/bin/bash", "-cex" ]

COPY go.mod go.sum /src/
RUN go mod download

COPY . /src/
RUN make build

FROM ${RELEASE_IMAGE} as release

RUN apt-get update && apt-get install -y iproute2 telnet net-tools bash-completion vim

LABEL org.opencontainers.image.authors='greenlab@jinr.ru' \
      org.opencontainers.image.url='https://dlnp.jinr.ru' \
      org.opencontainers.image.vendor='GreenLab' \
      org.opencontainers.image.licenses='Apache-2.0'


RUN echo 'source /etc/bash_completion' >> /etc/bash.bashrc \
  && echo 'source <(go-adc completion)' >> /etc/bash.bashrc

COPY --from=builder /src/bin/go-adc /usr/local/bin/go-adc

WORKDIR /data
