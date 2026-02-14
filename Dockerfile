FROM openresty/openresty:1.25.3.1-0-alpine-fat

RUN mkdir -p /usr/local/openresty/nginx/lua && \
    mkdir -p /usr/share/nginx/static_public && \
    mkdir -p /var/log/nginx

COPY ./nginx.conf /usr/local/openresty/nginx/conf/nginx.conf
COPY ./jwt-auth.lua /usr/local/openresty/nginx/lua/jwt-auth.lua
COPY ./jwt.lua /usr/local/openresty/lualib/resty/jwt.lua

COPY ./static_public /usr/share/nginx/static_public

RUN chown -R nobody:nobody /usr/share/nginx && \
    chown -R nobody:nobody /var/log/nginx && \
    chmod -R 755 /usr/share/nginx

EXPOSE 7000

CMD ["/usr/local/openresty/bin/openresty", "-g", "daemon off; error_log stderr info;"]