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


def print_help():
    print('strip_old_descriptions.py -i <inputfile>')


def parse_args():
    input_file = ''
    try:
        opts, args = getopt.getopt(sys.argv[1:], "hi:", ["ifile="])
    except getopt.GetoptError:
        print_help()
        sys.exit(2)

    for opt, arg in opts:
        if opt in ('-h', "--help"):
            print_help()
            sys.exit()
        elif opt in ("-i", "--ifile"):
            input_file = arg

    if input_file == '':
        print_help()
        sys.exit(2)

    return input_file


def main():
    input_file = parse_args()
    with open(input_file) as file:
        crd = yaml.load(file, Loader=yaml.SafeLoader)
        if crd and len(crd["spec"]["versions"]) > 1:
            crd["spec"]["versions"] = [
                v if v.get("storage", False) else remove_descriptions(v)
                for v in crd["spec"]["versions"]
            ]
            print(yaml.dump(crd), end='')


if __name__ == "__main__":
    main()
