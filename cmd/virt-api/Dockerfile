FROM centos:7

RUN curl -OL https://github.com/swagger-api/swagger-ui/tarball/38f74164a7062edb5dc80ef2fdddda24f3f6eb85/swagger-ui.tar.gz \
    && mkdir swagger-ui && tar xf swagger-ui.tar.gz -C swagger-ui --strip-components 1 \
    && mkdir /third_party \
    && mv swagger-ui/dist /third_party/swagger-ui && rm -rf swagger-ui \
    && sed -e 's@"http://petstore.swagger.io/v2/swagger.json"@"/swaggerapi/"@' -i  /third_party/swagger-ui/index.html \
    && rm swagger-ui.tar.gz && rm -rf swagger-ui

COPY virt-api /virt-api

ENTRYPOINT [ "/virt-api" ]
