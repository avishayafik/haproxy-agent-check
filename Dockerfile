FROM golang:1.15rc2-alpine3.12

ADD cpu.go /opt/cpu.go

WORKDIR /opt

RUN apk update && apk add --no-cache git  && go get github.com/firstrow/tcp_server github.com/mackerelio/go-osstat/cpu github.com/shirou/gopsutil/load gopkg.in/yaml.v2 github.com/prometheus/client_golang/api github.com/prometheus/client_golang/api/prometheus/v1 github.com/prometheus/client_golang/api github.com/prometheus/common/config && \

    adduser haproxy-agent --disabled-password --uid 1005 && \
    chown haproxy-agent.haproxy-agent /opt/cpu.go



USER 1005


CMD ["go" ,"run" , "/opt/cpu.go"] 
