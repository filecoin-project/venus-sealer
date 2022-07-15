FROM filvenus/venus-buildenv AS buildenv

COPY . ./venus-sealer
RUN export GOPROXY=https://goproxy.cn && cd venus-sealer  && make deps && make

RUN cd venus-sealer && ldd ./venus-sealer


FROM filvenus/venus-runtime

# DIR for app
WORKDIR /app

# copy the sealer app
COPY --from=buildenv  /go/venus-sealer/venus-sealer /app/venus-sealer



COPY ./docker/script  /script


EXPOSE 2345

ENTRYPOINT ["/script/init.sh"]
