FROM alpine

RUN mkdir -p /opt/resource
ADD ../check/check /opt/resource/check
ADD ../out/out /opt/resource/out
ADD ../in/in /opt/resource/in
