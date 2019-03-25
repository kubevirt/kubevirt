#!/bin/env python

import sys
import os
from jinja2 import Environment, FileSystemLoader

j2FilePath = sys.argv[1]
dockerPrefix = sys.argv[2]
operatorImage = sys.argv[3]
dockerTag = sys.argv[4]

controllerImage = sys.argv[5]
importerImage = sys.argv[6]
clonerImage = sys.argv[7]
apiserverImage = sys.argv[8]
uploadproxyImage = sys.argv[9]
uploadserverImage = sys.argv[10]

cdiNamespace = "cdi"
pullPolicy = "IfNotPresent"
verbosity = "1"


env = Environment(loader=FileSystemLoader(os.path.dirname(j2FilePath)),
                  trim_blocks=True)

template = env.get_template("cdi-operator.yaml.j2")
rendered_template = template.render(docker_prefix=dockerPrefix,
                                    operator_image=operatorImage,
                                    docker_tag=dockerTag,
                                    controller_image=controllerImage,
                                    importer_image=importerImage,
                                    cloner_image=clonerImage,
                                    apiserver_image=apiserverImage,
                                    uploadproxy_image=uploadproxyImage,
                                    uploadserver_image=uploadserverImage,

                                    cdi_namespace=cdiNamespace,
                                    pull_policy=pullPolicy,
                                    verbosity=verbosity
                                    )

cdi = open(os.path.dirname(j2FilePath) + "/cdi-operator.yaml", "w")
cdi.write(rendered_template)
cdi.close()
