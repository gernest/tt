FROM alpine:3.14
COPY tt /usr/local/bin/tt
ENTRYPOINT [ "/usr/local/bin/tt" ]