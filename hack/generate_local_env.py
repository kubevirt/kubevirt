#! /usr/bin/python

import sys
import re
from os import environ, linesep

CSV_VERSION = '1.4.0'
KUBEVIRT_CLIENT_GO_SCHEME_REGISTRATION_VERSION = 'v1'

def get_env(line):
    env = line[:line.find('=')]
    return f'{env}={environ.get(env)}'


def get_env_file(outdir, frmt='txt'):
    rgx = re.compile('^[^ #]+=.*$')
    if frmt == 'env':
        sep = linesep
        ext = '.env'
    else:
        sep = ';'
        ext = '.txt'

    with open('hack/config') as infile:
        with open(f'{outdir}/envs{ext}', 'w') as out:
            vars = [line.strip() for line in infile if rgx.match(line)]
            vars.append('KUBECONFIG=None')
            vars.append('WEBHOOK_MODE=false')
            vars = map(lambda s: get_env(s), vars)
            var_str = f"{sep.join(vars)}{sep}WATCH_NAMESPACE=kubevirt-hyperconverged{sep}OSDK_FORCE_RUN_MODE=local{sep}OPERATOR_NAMESPACE=kubevirt-hyperconverged"
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
