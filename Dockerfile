FROM golang:1.6.0

RUN apt-get update && \
    apt-get install -y --no-install-recommends libvips-dev libgsf-1-dev

RUN go get github.com/shifr/imgwizard

ENV PATH $PATH:/usr/local/go/bin
ENV PKG_CONFIG_PATH /usr/local/lib/pkgconfig:/usr/lib/pkgconfig

EXPOSE 8070

ENTRYPOINT imgwizard -l 0.0.0.0:8070
