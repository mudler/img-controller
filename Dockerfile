FROM quay.io/mocaccino/extra as builder

ADD . /img-controller
RUN luet install -y lang/go
RUN cd /img-controller && CGO_ENABLED=0 go build

FROM ubuntu:20.04
RUN apt-get update
ENV LUET_YES=true

RUN apt-get install -y uidmap curl wget git libcap2-bin
RUN curl https://get.mocaccino.org/luet/get_luet_root.sh | sh
RUN luet install repository/mocaccino-extra && luet install container/img && luet upgrade
RUN chmod u-s /usr/bin/new[gu]idmap && \
    setcap cap_setuid+eip /usr/bin/newuidmap && \
    setcap cap_setgid+eip /usr/bin/newgidmap 

RUN mkdir -p /run/runc  && chmod 777 /run/runc

RUN useradd -u 1000 -d /img -ms /bin/bash img
USER img
WORKDIR /luet
COPY --from=builder /img-controller/img-controller /usr/bin/img-controller
RUN chmod -R 777 /luet
ENTRYPOINT "/usr/bin/img-controller"
