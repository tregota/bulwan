FROM scratch
ARG default_url=""
ENV SERVER_HOST=$default_url
ENV SERVER_PORT=22
ENV SERVER_PUBLIC_KEY=$default_url
ENV SERVER_PUBLIC_KEY_TYPE=$default_url
ENV SSH_USERNAME=$default_url
ENV SSH_LISTEN_PORT=$default_url
ENV SSH_PRIVATE_KEY=$default_url
ENV HTTP_GET_ON_CLOSE=$default_url
ENV EXPOSED_HTTP_SERVERS_PREFIX_1="routerprefix"
ENV EXPOSED_HTTP_SERVERS_URL_1="http://host:port"
ENV EXPOSED_HTTP_SERVERS_PREFIX_2=$default_url
ENV EXPOSED_HTTP_SERVERS_URL_2=$default_url
EXPOSE 35300/tcp
ADD main /
ADD settings.conf /
CMD ["/main"]