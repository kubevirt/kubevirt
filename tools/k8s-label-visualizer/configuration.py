#! /usr/bin/python
import json
from dataclasses import dataclass
from typing import List, Optional

import marshmallow_dataclass


@dataclass
class GroupVersionKind(object):
    group: str
    version: str
    kind: str


@dataclass
class Configuration(object):
    namespaced: Optional[List[GroupVersionKind]]
    cluster: Optional[List[GroupVersionKind]]
    alias: Optional[dict]
    component_alias: Optional[dict]

    @classmethod
    def load_from_json_file(cls, filename):
        with open(filename, 'r') as conf_file:
            data = conf_file.read()

        conf_schema = marshmallow_dataclass.class_schema(Configuration)()
        return conf_schema.load(json.loads(data))
