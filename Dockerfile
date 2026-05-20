FROM scratch

COPY build/docker/p2pnode /p2pnode
EXPOSE 8080 9000
ENTRYPOINT ["/p2pnode"]
