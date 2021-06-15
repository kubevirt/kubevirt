#!/usr/bin/env python3
import yaml
import getopt
import sys


def remove_descriptions(obj):
    rubbish = ["description"]
    if isinstance(obj, dict):
        obj = {
            key: remove_descriptions(value)
            for key, value in obj.items()
            if key not in rubbish
        }
    elif isinstance(obj, list):
        obj = [
            remove_descriptions(item)
            for item in obj
            if item not in rubbish
        ]
    return obj


def main():
    inputfile = ''
    try:
        opts, args = getopt.getopt(sys.argv[1:], "hi:", ["ifile="])
    except getopt.GetoptError:
        print('strip_old_descriptions.py -i <inputfile>')
        sys.exit(2)
    for opt, arg in opts:
        if opt == '-h':
            print('strip_old_descriptions.py -i <inputfile>')
            sys.exit()
        elif opt in ("-i", "--ifile"):
            inputfile = arg
    with open(inputfile) as file:
        crd = yaml.load(file, Loader=yaml.SafeLoader)
        if crd:
            crd_versions = []
            if len(crd["spec"]["versions"]) > 1:
                for v in crd["spec"]["versions"]:
                    if not v.get("storage", False):
                        crd_versions.append(remove_descriptions(v))
                    else:
                        crd_versions.append(v)
                crd["spec"]["versions"] = crd_versions
                print(yaml.dump(crd), end='')


if __name__ == "__main__":
    main()
