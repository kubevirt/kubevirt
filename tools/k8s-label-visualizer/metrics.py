import json
import os
import subprocess

import configuration

MEMORY_QUERY = 'sum by (label_app_kubernetes_io_component) ' \
               '(sum(container_memory_usage_bytes{' \
               'namespace="kubevirt-hyperconverged"}) by (pod) * ' \
               'on (pod) group_left(label_app_kubernetes_io_component) ' \
               'kube_pod_labels{namespace="kubevirt-hyperconverged"})' \
               ' / (1024* 1024)'
CPU_QUERY = 'sum by (label_app_kubernetes_io_component) ' \
            '(sum(pod:container_cpu_usage:sum{' \
            'namespace="kubevirt-hyperconverged"}) by (pod) * ' \
            'on (pod) group_left(label_app_kubernetes_io_component) ' \
            'kube_pod_labels{namespace="kubevirt-hyperconverged"})'

METRIC_LABEL_FOR_COMPONENT = "label_app_kubernetes_io_component"


class GraphGenerator(object):

    def __init__(self, conf: configuration.Configuration, outdir):
        self.conf = conf
        self.outdir = outdir

    def generate(self):
        output_file = os.path.join(self.outdir, "metrics.txt")
        with open(output_file, "w") as file:
            print_to_file(
                file,
                "MEMORY CONSUMPTION",
                self.run_prometheus_query(MEMORY_QUERY)
            )
            print_to_file(
                file,
                "CPU CONSUMPTION",
                self.run_prometheus_query(CPU_QUERY)
            )

    def run_prometheus_query(self, prometheus_query):
        oc_command = "oc exec -n openshift-monitoring prometheus-k8s-0 " \
                     "-c prometheus -- " \
                     "curl --silent --data-urlencode 'query={}' " \
                     "http://127.0.0.1:9090/api/v1/query".format(
                        prometheus_query
                     )
        try:
            result_as_json_string = subprocess.check_output(
                oc_command, shell=True, stderr=subprocess.STDOUT
            )
        except subprocess.CalledProcessError:
            raise RuntimeError(
                "Result for prometheus query is not success."
            )
        result = json.loads(result_as_json_string.decode('UTF-8'))

        return self.convert_to_dic_per_component(result['data']['result'])

    def convert_to_dic_per_component(self, raw_dict):
        """
        Iterate over raw prometheus query result
        and convert it to a simple dictionary
        :param raw_dict: .data.result part
        of prometheus query result
        :return: a dictionary with component names
        as keys and query results as values
        """
        return dict(
            (
                i.get('metric').get(
                    METRIC_LABEL_FOR_COMPONENT
                ) or 'unassigned',
                i.get('value')[1]
            )
            for i in raw_dict
        )

    def transform_component_alias(self, component_name):
        return self.conf.component_alias.get(component_name) or component_name


def print_to_file(file, title, data):
    file.write(title)
    file.write("\n")
    for item in data:
        file.write("{} : {}\n".format(item, data[item]))

    file.write("\n\n\n")
