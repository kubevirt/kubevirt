#! /usr/bin/python3

import argparse

import dot_graph
import metrics
from configuration import Configuration
from k8s_fetcher import ObjectFetcher

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        '--namespace',
        type=str,
        dest='namespace',
        help="k8s namespace",
        required=True
    )
    parser.add_argument(
        '--conf',
        type=str,
        dest='conf',
        help="configuration file",
        required=True
    )
    parser.add_argument(
        '--output',
        type=str,
        dest='out',
        help="output directory"
    )
    args = parser.parse_args()

    conf = Configuration.load_from_json_file(args.conf)

    fetcher = ObjectFetcher(conf)
    objects = fetcher.fetch(args.namespace)

    generator = dot_graph.GraphGenerator(conf, objects, args.out)
    generator.generate()

    try:
        metric_generator = metrics.GraphGenerator(conf, args.out)
        metric_generator.generate()
    except RuntimeError:
        print("Failed connecting metrics")
