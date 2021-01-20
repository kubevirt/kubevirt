#! /usr/bin/python

import sys
import re
from os import environ, linesep

CSV_VERSION = '1.4.0'
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

    with open('hack/config') as infile:
        with open(f'{outdir}/envs{ext}', 'w') as out:
            vars_list = [line.strip() for line in infile if rgx.match(line)]
            vars_list.append('KUBECONFIG=None')
            vars_list.append('WEBHOOK_MODE=false')
            vars_list = map(lambda s: get_env(s), vars_list)
            var_str = f"{sep.join(vars_list)}{sep}WATCH_NAMESPACE=kubevirt-hyperconverged{sep}OSDK_FORCE_RUN_MODE=local{sep}OPERATOR_NAMESPACE=kubevirt-hyperconverged"
            var_str = var_str + f"{sep}WEBHOOK_CERT_DIR=./_local/certs"
            var_str = var_str + f"{sep}HCO_KV_IO_VERSION={CSV_VERSION}"
            var_str = var_str + f"{sep}KUBEVIRT_CLIENT_GO_SCHEME_REGISTRATION_VERSION={KUBEVIRT_CLIENT_GO_SCHEME_REGISTRATION_VERSION}"
            var_str = var_str.replace("CONVERSION_CONTAINER_VERSION=", "CONVERSION_CONTAINER=").replace("VMWARE_CONTAINER_VERSION=", "VMWARE_CONTAINER=")
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
