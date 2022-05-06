FROM filvenus/venus-buildenv AS buildenv

RUN git clone https://github.com/filecoin-project/venus-sealer.git --depth 1 
RUN export GOPROXY=https://goproxy.cn && cd venus-sealer  && make deps && make

RUN cd venus-sealer && ldd ./venus-sealer


FROM filvenus/venus-runtime

# DIR for app
WORKDIR /app

# 拷贝当前目录下 可以执行文件
COPY --from=buildenv  /go/venus-sealer/venus-sealer /app/venus-sealer


# 拷贝依赖库
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

# 暴露端口
EXPOSE 2345

# 运行golang程序的命令
ENTRYPOINT ["/script/init.sh"]
# ENTRYPOINT ["/bin/bash"]


