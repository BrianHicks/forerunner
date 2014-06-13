FROM busybox

ADD forerunner /usr/local/bin/forerunner
RUN chmod +x /usr/local/bin/forerunner

ENTRYPOINT /usr/local/bin/forerunner
CMD ["--help"]
