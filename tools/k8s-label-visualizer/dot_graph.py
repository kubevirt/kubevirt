from graphviz import Digraph

import configuration

KUBERNETES_IO_COMPONENT = "app.kubernetes.io/component"
KUBERNETES_IO_MANAGED_BY = "app.kubernetes.io/managed-by"


def get_labels(i):
    if "metadata" in i and "labels" in i["metadata"]:
        return i["metadata"]["labels"]
    return []


class GraphGenerator(object):

    def __init__(self, conf: configuration.Configuration, objects, outdir):
        self.conf = conf
        self.objects = objects
        self.outdir = outdir

    def generate(self):
        self.generate_component_graph()
        self.generate_managed_by_graph()

    def generate_managed_by_graph(self):
        dot = Digraph(
            name="managed-by",
            graph_attr={
                'rankdir': 'LR',
                'center': 'true',
                'margin': '0.1',
                'nodesep': '0.1',
                'ranksep': '2'
            },
            node_attr={
                'shape': 'box',
                'style': 'rounded',
                'fontname': 'Courier-Bold',
                'fontsize': '24',
                'width': '3',
                'height': '1'
            },
            edge_attr={
                'arrowsize': '2',
                'arrowhead': 'vee'
            },
            format='svg',
        )

        for i in self.objects:
            node_name = self.get_node_name(i)
            dot.node(node_name, node_name)

            labels = get_labels(i)
            if KUBERNETES_IO_MANAGED_BY in labels:
                managed_node_name = self.transform_alias(
                    labels[KUBERNETES_IO_MANAGED_BY]
                )
                dot.node(managed_node_name, managed_node_name)
                dot.edge(managed_node_name, node_name)

        self.render_graph(dot, "managed-by.gv")

    def generate_component_graph(self):
        dot = Digraph(
            name="component",
            graph_attr={'rankdir': 'LR'},
            node_attr={
                'shape': 'box',
                'style': 'rounded',
                'fontname': 'Courier-Bold',
                'fontsize': '24',
                'width': '3',
                'height': '1'
            },
            format='svg',
        )

        component_dict = dict()
        for i in self.objects:
            node_name = self.get_node_name(i)

            labels = get_labels(i)
            if KUBERNETES_IO_COMPONENT in labels:
                component_name = self.transform_component_alias(
                    labels[KUBERNETES_IO_COMPONENT]
                )
            else:
                component_name = "unassigned"

            if component_name not in component_dict:
                component_dict[component_name] = []
            component_dict[component_name].append(node_name)

        for s_k, s_v in sorted(component_dict.items()):
            with dot.subgraph(
                name='cluster_' + s_k
            ) as c:
                c.attr(
                    label=s_k,
                    rank="same",
                    style="dashed",
                    group=s_k,
                    fontname="Courier-Bold",
                    fontsize="42"
                )
                for v in sorted(s_v):
                    c.node(v, v)

        dot.attr('edge', style='invis')
        prev = None
        for s_k, s_v in sorted(component_dict.items()):
            sorted_l = sorted(s_v)
            if prev:
                dot.edge(prev, sorted_l[0])
            prev = sorted_l[0]

        self.render_graph(dot, "component.gv")

    def render_graph(self, g, name):
        g.render(directory=self.outdir, filename=name)

    def get_node_name(self, i):
        node_name = i["kind"] + "/" + i["metadata"]["name"]
        return self.transform_alias(node_name)

    def transform_alias(self, node_name):
        return self.conf.alias.get(node_name) or node_name

    def transform_component_alias(self, component_name):
        return self.conf.component_alias.get(component_name) or component_name
