FROM golang:latest as builder
COPY .. /src/storage
RUN go env -w GOPROXY=https://goproxy.cn,direct && cd /src/storage && make all &&  \
    /bin/cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && echo 'Asia/Shanghai' > /etc/timezone

FROM golang:latest as prod
COPY --from=builder /src/storage /storage
WORKDIR /storage
ENTRYPOINT [ "./bin/storage"]
EXPOSE 8888

