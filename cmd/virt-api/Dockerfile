FROM centos:7

RUN curl -OL https://github.com/swagger-api/swagger-ui/archive/v2.2.6.tar.gz && tar xf v2.2.6.tar.gz \
    && mkdir /third_party \
    && mv swagger-ui-2.2.6/dist /third_party/swagger-ui && rm -rf swagger-ui-2.2.6 \
    && sed -e 's@url = "http://petstore.swagger.io/v2/swagger.json";@url = "/apidocs.json";@' -i  /third_party/swagger-ui/index.html \
    && rm v2.2.6.tar.gz && rm -rf swagger-ui*


COPY virt-api /virt-api

ENTRYPOINT [ "/virt-api" ]
