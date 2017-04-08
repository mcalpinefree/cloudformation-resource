FROM alpine

RUN mkdir -p /opt/resource
ADD ./check/check /opt/resource/
ADD ./out/out /opt/resource/
ADD ./in/in /opt/resource/
