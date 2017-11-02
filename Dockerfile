FROM alpine

RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
RUN apk update && apk add ca-certificates
RUN mkdir -p /opt/resource
ADD ./check/check /opt/resource/
ADD ./out/out /opt/resource/
ADD ./in/in /opt/resource/
