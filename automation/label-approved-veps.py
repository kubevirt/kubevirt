import os
import re
import requests

GITHUB_TOKEN = os.environ["GITHUB_TOKEN"]
KUBEVIRT_REPO = os.environ.get("GITHUB_REPOSITORY", "kubevirt/kubevirt")
PR_NUMBER = os.environ["PR_NUMBER"]
TARGET_PROJECT_URL = os.environ.get("TARGET_PROJECT_URL")
HEADERS = {
    "Authorization": f"token {GITHUB_TOKEN}",
    "Accept": "application/vnd.github.v3+json",
}
GRAPHQL_API_URL = "https://api.github.com/graphql"


def get_pr_details():
    """Fetch the kubevirt/kubevirt PR body."""
    url = f"https://api.github.com/repos/{KUBEVIRT_REPO}/pulls/{PR_NUMBER}"
    response = requests.get(url, headers=HEADERS)
    response.raise_for_status()
    return response.json()["body"]


def extract_enhancements_references(pr_body):
    """Get the enhancements reference numbers from the PR."""
    # Regex to find enhancement issue/pull numbers
    pattern = (
        r"(?:https://github.com/)?kubevirt/enhancements/(?:issues|pull)/(\d+)"
        r"|(?:kubevirt/)?enhancements#(\d+)"
    )
    matches = re.findall(pattern, pr_body)

    # Extract the first non-empty group from each match
    ref_numbers = {
        group[0] or group[1] for group in matches if group[0] or group[1]
    }
    return list(ref_numbers)


def add_label_to_pr():
    """Add the 'approved-vep' label to the kubevirt PR."""
    url = (
        f"https://api.github.com/repos/{KUBEVIRT_REPO}/issues/"
        f"{PR_NUMBER}/labels"
    )
    payload = {"labels": ["approved-vep"]}
    response = requests.post(url, headers=HEADERS, json=payload)
    response.raise_for_status()


def parse_project_url(project_url):
    """
    Extract the project number from the GitHub project URL.
    For example: https://github.com/orgs/kubevirt/projects/15
    """
    match = re.match(
                r"https://github.com/orgs/kubevirt/projects/(\d+)",
                project_url)
    if match:
        return int(match.group(1))

    msg_part1 = f"Invalid project URL format for kubevirt org: {project_url}."
    msg_part2 = ("Expected format: https://github.com/orgs/kubevirt/"
                 "projects/PROJECT_NUMBER")
    raise ValueError(f"{msg_part1} {msg_part2}")


def execute_graphql_query(query, variables):
    """Execute a GraphQL query."""
    try:
        response = requests.post(
            GRAPHQL_API_URL,
            headers=HEADERS,
            json={"query": query, "variables": variables},
        )
        response.raise_for_status()
        return response.json()
    except requests.exceptions.RequestException as e:
        print(f"GraphQL request failed: {e}")
        return None


def get_tracked_enhancement_issues_from_project(project_number):
    """
    Get VEP issue numbers from kubevirt/enhancements that are tracked
    in the target project and are approved.

    A "Tracked" issue means that it is approved for the release.
    """
    tracked_issues = {}

    query = """
    query($orgName: String!, $projectNumber: Int!, $cursor: String) {
      organization(login: $orgName) {
        projectV2(number: $projectNumber) {
          items(first: 100, after: $cursor) {
            nodes {
              id
              content {
                __typename
                ... on Issue {
                  number
                  repository {
                    nameWithOwner
                  }
                }
              }
              fieldValues(first: 20) {
                nodes {
                  __typename
                  ... on ProjectV2ItemFieldSingleSelectValue {
                    selected_option_name: name
                    field {
                      field_actual_typename: __typename
                      ... on ProjectV2SingleSelectField {
                          field_definition_id: id
                          field_definition_name: name
                      }
                    }
                  }
                }
              }
            }
            pageInfo {
              hasNextPage
              endCursor
            }
          }
        }
      }
    }
    """
    variables = {"orgName": "kubevirt", "projectNumber": project_number}
    has_next_page = True
    current_cursor = None

    print(
        f"Fetching items from project 'kubevirt/projects/{project_number}' "
        "to check for 'Status: Tracked'"
    )
    while has_next_page:
        variables["cursor"] = current_cursor
        result = execute_graphql_query(query, variables)

        if not result or "errors" in result:
            error_payload = result.get('errors') if result else 'No response'
            print(f"GraphQL query errors: {error_payload}")
            break

        data = result.get("data", {})
        organization = data.get("organization", {})
        project_data = organization.get("projectV2", {})

        if not project_data or not project_data.get("items"):
            print(
                f"Warning: No items found for kubevirt project number "
                f"{project_number}."
            )
            break

        items_data = project_data.get("items", {})
        nodes = items_data.get("nodes", [])

        for item_node in nodes:
            content = item_node.get("content")

            if not content:
                continue
            if content.get("__typename") != "Issue":
                continue
            repo_info = content.get("repository", {})
            if repo_info.get("nameWithOwner") != "kubevirt/enhancements":
                continue
            if "number" not in content:
                continue

            issue_number = content["number"]
            is_tracked = False
            field_values_data = item_node.get("fieldValues", {})
            field_value_nodes = field_values_data.get("nodes", [])

            for field_value_node in field_value_nodes:
                expected_type = "ProjectV2ItemFieldSingleSelectValue"
                if field_value_node.get("__typename") == expected_type:
                    field = field_value_node.get("field", {})
                    actual_field_name = field.get("field_definition_name")
                    is_status_field = actual_field_name == "Status"
                    is_tracked_value = (
                        field_value_node.get(
                            "selected_option_name") == "Tracked"
                    )
                    if is_status_field and is_tracked_value:
                        is_tracked = True
                        break

            if is_tracked:
                tracked_issues[issue_number] = True

        page_info = items_data.get("pageInfo", {})
        has_next_page = page_info.get("hasNextPage", False)
        current_cursor = page_info.get("endCursor") if has_next_page else None

    return tracked_issues


def main():
    """Main execution function."""
    try:
        project_number = parse_project_url(TARGET_PROJECT_URL)
    except ValueError as e:
        print(f"Error parsing project URL '{TARGET_PROJECT_URL}':\n{e}")
        return

    pr_body = get_pr_details()
    if not pr_body:
        print("No PR body found.")
        return

    ref_numbers_str = extract_enhancements_references(pr_body)
    if not ref_numbers_str:
        print("No enhancements references found.")
        return

    ref_numbers = {int(num_str) for num_str in ref_numbers_str}
    print(
        f"Referenced KubeVirt Enhancement issue(s) in PR "
        f"{KUBEVIRT_REPO}/pulls/{PR_NUMBER}: {ref_numbers}"
    )

    tracked_issues_in_project = get_tracked_enhancement_issues_from_project(
        project_number
    )

    first_matching_vep = None
    for vep_issue_num in ref_numbers:
        if vep_issue_num in tracked_issues_in_project:
            first_matching_vep = vep_issue_num
            break

    if first_matching_vep is not None:
        print(
            f"Match: KubeVirt Enhancement issue #{first_matching_vep} "
            f"(referenced in this PR) is tracked in project "
            f"{TARGET_PROJECT_URL}."
        )
        print(
            f"This PR ({KUBEVIRT_REPO}/pulls/{PR_NUMBER}) is related to a VEP "
            "tracked for the current release."
        )
        add_label_to_pr()
    else:
        print(
            f"This PR ({KUBEVIRT_REPO}/pulls/{PR_NUMBER}) is not related to "
            f"any VEP issue currently tracked in project {TARGET_PROJECT_URL}."
        )


if __name__ == "__main__":
    main()
