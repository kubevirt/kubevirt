#! /usr/bin/python

import yaml
import sys

CSV_VERSION = '1.2.0'


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
            # del c['spec']['template']['spec']['serviceAccountName']

        out.write('\n---\n'.join(map(lambda x: yaml.dump(x), csv)))


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("one argument of output dir is required")
        exit(-1)
    gen_local_deployments(sys.argv[1])
