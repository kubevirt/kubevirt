import re
import sys

wrong_len_equal_check = "(Expect(?:\\(|WithOffset\\(\\d+, ?))len\\(([^)]+)\\)(\\)\\.(?:To|ToNot|NotTo|Should|" \
                        "ShouldNot)\\((?:Not\\()?)Equal\\(([^\\)]+)\\)(.*$)"
wrong_len_equal_regex = re.compile(wrong_len_equal_check)

wrong_len_zero_check = "(Expect(?:\\(|WithOffset\\(\\d+, ?))len\\(([^)]+)\\)(\\)\\.(?:To|ToNot|NotTo|Should|" \
                       "ShouldNot)\\((?:Not\\()?)BeZero\\(\\)(.*)$"
wrong_len_zero_regex = re.compile(wrong_len_zero_check)

wrong_empty_check = "(Expect(?:\\(|WithOffset\\(\\d+, ?))len\\(([^)]+)\\)(\\)\\.(?:To|ToNot|NotTo|Should|ShouldNot)" \
                    "\\((?:Not\\()?)BeNumerically\\((?:\">\", 0|\">=\", 1)\\)(.*)$"
wrong_empty_regex = re.compile(wrong_empty_check)


def find_wrong_len_equal_check(file_name, line_number, line):
    res = wrong_len_equal_regex.search(line)
    if res:
        matcher = "BeEmpty()" if res[4].isnumeric() and int(res[4]) == 0 else f"HaveLen({res[4]})"
        use = f'{res[1]}{res[2]}{res[3]}{matcher}{res[5]}'
        wrong_length_output(file_name, line, line_number, use)
        return 1
    return 0


def find_wrong_len_zero_check(file_name, line_number, line):
    res = wrong_len_zero_regex.search(line)
    if res:
        use = f"{res[1]}{res[2]}{res[3]}BeEmpty(){res[4]}"
        wrong_length_output(file_name, line, line_number, use)
        return 1
    return 0


def find_wrong_empty_check(file_name, line_number, line):
    res = wrong_empty_regex.search(line)
    if res:
        use = f"{res[1]}{res[2]}{res[3]}Not(BeEmpty()){res[4]}"
        wrong_length_output(file_name, line, line_number, use)
        return 1
    return 0


checks = [find_wrong_len_equal_check, find_wrong_len_zero_check, find_wrong_empty_check]


def check_one_file(file_name):
    found = 0
    with open(file_name) as f:
        i = 0
        for line in f:
            i = i + 1
            line = line.strip()
            found = found + sum([func(file_name, i, line) for func in checks])
    return found


def wrong_length_output(file_name, line, line_num, use):
    print(f'''
Found issue in {file_name}, line #{line_num}: wrong length check:
    {line}
Consider replacing with: 
    {use}
{"=" * 80}''', file=sys.stderr)


def main():
    found_issues = 0
    found_files = 0
    if len(sys.argv) > 1:
        for file in sys.argv[1:]:
            found_in_one_file = check_one_file(file)
            if found_in_one_file > 0:
                found_issues = found_issues + found_in_one_file
                found_files = found_files + 1
    if found_issues > 0:
        print(f'Found {found_issues} issue{"s" if found_issues > 1 else ""} in '
              f'{found_files} file{"s" if found_files > 1 else ""}', file=sys.stderr)
        sys.exit(1)

    print('Success: ginkgo-linter found no issues.')


if __name__ == '__main__':
    main()
