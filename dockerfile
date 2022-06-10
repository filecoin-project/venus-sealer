FROM filvenus/venus-buildenv AS buildenv

COPY . ./venus-sealer
RUN export GOPROXY=https://goproxy.cn && cd venus-sealer  && make deps && make

RUN cd venus-sealer && ldd ./venus-sealer


FROM filvenus/venus-runtime

# DIR for app
WORKDIR /app

# copy the sealer app
COPY --from=buildenv  /go/venus-sealer/venus-sealer /app/venus-sealer


# copy ddl
COPY --from=buildenv   /usr/lib/x86_64-linux-gnu/libhwloc.so.5  \
    /usr/lib/x86_64-linux-gnu/libOpenCL.so.1  \
    /lib/x86_64-linux-gnu/libgcc_s.so.1  \
    /lib/x86_64-linux-gnu/libutil.so.1  \
    /lib/x86_64-linux-gnu/librt.so.1  \
    /lib/x86_64-linux-gnu/libpthread.so.0  \
    /lib/x86_64-linux-gnu/libm.so.6  \
    /lib/x86_64-linux-gnu/libdl.so.2  \
    /lib/x86_64-linux-gnu/libc.so.6  \
    /usr/lib/x86_64-linux-gnu/libnuma.so.1  \
    /usr/lib/x86_64-linux-gnu/libltdl.so.7  \
    /lib/

COPY ./docker/script  /script


EXPOSE 2345

ENTRYPOINT ["/script/init.sh"]
