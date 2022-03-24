import urllib3
from kubernetes import config, client

import configuration


class ObjectFetcher(object):

    def __init__(self, conf: configuration.Configuration):
        self.conf = conf

    def fetch(self, ns):
        urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
        try:
            config.load_incluster_config()
        except config.ConfigException:
            try:
                config.load_kube_config()
            except config.ConfigException:
                raise Exception("Could not configure kubernetes python client")
        api = client.CustomObjectsApi()

        result = []
        for nt in self.conf.namespaced:
            response = api.list_namespaced_custom_object(
                namespace=ns,
                group=nt.group,
                version=nt.version,
                plural=nt.kind,
                watch=False
            )
            result.extend(response["items"])

        for ct in self.conf.cluster:
            response = api.list_cluster_custom_object(
                group=ct.group,
                version=ct.version,
                plural=ct.kind,
                watch=False
            )
            result.extend(response["items"])

        v1 = client.AppsV1Api()
        deployments = v1.list_namespaced_deployment(ns)
        for d in deployments.items:
            d.kind = "deployment"
            result.append(d.to_dict())

        ds = v1.list_namespaced_daemon_set(ns)
        for d in ds.items:
            d.kind = "daemonset"
            result.append(d.to_dict())

        return result
