#! /usr/bin/python

import sys
import re
from os import environ, linesep
import csv

KUBEVIRT_CLIENT_GO_SCHEME_REGISTRATION_VERSION = 'v1'

def get_env(line):
    env = line[:line.find('=')]
    return f'{env}={environ.get(env)}'


def get_env_file(outdir, file_format='txt'):
    rgx = re.compile('^[^ #]+=.*$')
    if file_format == 'env':
        sep = linesep
        ext = '.env'
    else:
        sep = ';'
        ext = '.txt'

    vars_list = ""
    with open('hack/config') as infile:
        vars_list = [line.strip() for line in infile if rgx.match(line)]
    vars_list.append("KUBECONFIG=None")

    vars_list = list(map(lambda s: get_env(s), vars_list))

    with open('deploy/images.csv') as image_file:
        reader = csv.DictReader(image_file, delimiter=',')
        for row in reader:
            if row['image_var'] in ['VMWARE_IMAGE', 'CONVERSION_IMAGE', 'KUBEVIRT_VIRTIO_IMAGE']:
                image = f"{row['name']}@sha256:{row['digest']}"
                env = 'VIRTIOWIN_CONTAINER' if row['image_var'] == 'KUBEVIRT_VIRTIO_IMAGE' else row['image_var'].replace("_IMAGE", "_CONTAINER")
                vars_list.append(f"{env}={image}")

    vars_list.extend([
        f"HCO_KV_IO_VERSION={environ.get('CSV_VERSION')}",
        "WEBHOOK_MODE=false",
        "WEBHOOK_CERT_DIR=./_local/certs",
        f"KUBEVIRT_CLIENT_GO_SCHEME_REGISTRATION_VERSION={KUBEVIRT_CLIENT_GO_SCHEME_REGISTRATION_VERSION}",
        "WATCH_NAMESPACE=kubevirt-hyperconverged",
        "OSDK_FORCE_RUN_MODE=local",
        "OPERATOR_NAMESPACE=kubevirt-hyperconverged",
    ])

    var_str = sep.join(vars_list)

    with open(f'{outdir}/envs{ext}', 'w') as out:
        out.write(var_str)


if __name__ == "__main__":
    frmt = 'txt'
    if len(sys.argv) == 3:
        frmt = sys.argv[2]
    else:
        if len(sys.argv) != 2:
            print("one argument of output dir is required. The second argument is optional: 'env' for .env file format")
            exit(-1)

    get_env_file(sys.argv[1], frmt)
