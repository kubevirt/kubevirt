#! /usr/bin/python

import yaml
import sys
import re
from os import environ, linesep

CSV_VERSION = '1.3.0'


def get_env(line):
    env = line[:line.find('=')]
    return f'{env}={environ.get(env)}'


def gen_local_deployments(outdir):
    with open(f'deploy/olm-catalog/kubevirt-hyperconverged/{CSV_VERSION}/kubevirt-hyperconverged-operator.v{CSV_VERSION}.clusterserviceversion.yaml') as csv_file:
        csv = yaml.safe_load(csv_file)

    with open(f'{outdir}/local.yaml', 'w') as out:
        csv = [x for x in csv['spec']['install']['spec']['deployments'] if x['name'] != 'hco-operator']
        for c in csv:
            c.update({
                'apiVersion': 'apps/v1',
                'kind': 'Deployment',
                'metadata': {
                    'name': c['name'] + '-deployment'
                }
            })
            del(c['name'])

        out.write('\n---\n'.join(map(lambda x: yaml.dump(x), csv)))


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
            vars = map(lambda s: get_env(s), vars)
            var_str = f"{sep.join(vars)}{sep}WATCH_NAMESPACE=kubevirt-hyperconverged{sep}OSDK_FORCE_RUN_MODE=local{sep}OPERATOR_NAMESPACE=kubevirt-hyperconverged"
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

    gen_local_deployments(sys.argv[1])
    get_env_file(sys.argv[1], frmt)
