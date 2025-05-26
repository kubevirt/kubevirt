import os
import re
import requests

GITHUB_TOKEN = os.environ["GITHUB_TOKEN"]
KUBEVIRT_REPO = os.environ.get("GITHUB_REPOSITORY", "kubevirt/kubevirt")
PR_NUMBER = os.environ["PR_NUMBER"]
TARGET_PROJECT_URL = os.environ.get("TARGET_PROJECT_URL")
HEADERS = {"Authorization": f"token {GITHUB_TOKEN}", "Accept": "application/vnd.github.v3+json"}
GRAPHQL_API_URL = "https://api.github.com/graphql"


def get_pr_details():
    #Fetch the kubevirt/kubevirt PR body
    url = f"https://api.github.com/repos/{KUBEVIRT_REPO}/pulls/{PR_NUMBER}"
    response = requests.get(url, headers=HEADERS)
    response.raise_for_status()
    return response.json()["body"]

def extract_enhancements_references(pr_body):
    #Get the enhancements reference numbers from the PR.
    pattern = r"(?:https://github.com/)?kubevirt/enhancements/(?:issues|pull)/(\d+)|(?:kubevirt/)?enhancements#(\d+)"
    matches = re.findall(pattern, pr_body)

    # Extract the first non-empty group from each match and ensure they are issue numbers
    ref_numbers = {group[0] or group[1] for group in matches if group[0] or group[1]}
    return list(ref_numbers)

def add_label_to_pr():
    # Adds the 'approved-vep' label to the kubevirt PR. (when we will have that )
    url = f"https://api.github.com/repos/{KUBEVIRT_REPO}/issues/{PR_NUMBER}/labels"
    payload = {"labels": ["approved-vep"]}
    response = requests.post(url, headers=HEADERS, json=payload)
    response.raise_for_status()

def parse_project_url(project_url):
    # Extract the project number from the GitHub project URL.
    # For example: https://github.com/orgs/kubevirt/projects/15
    match = re.match(r"https://github.com/orgs/kubevirt/projects/(\d+)", project_url)
    if match:
        return int(match.group(1))
    raise ValueError(f"Invalid project URL format for kubevirt org: {project_url}. Expected format: https://github.com/orgs/kubevirt/projects/PROJECT_NUMBER")

def execute_graphql_query(query, variables):
    # Executes a GraphQL query.
    try:
        response = requests.post(GRAPHQL_API_URL, headers=HEADERS, json={"query": query, "variables": variables})
        response.raise_for_status()
        return response.json()
    except requests.exceptions.RequestException as e:
        print(f"GraphQL request failed: {e}")
        return None

def get_tracked_enhancement_issues_from_project(project_number):
    # Get VEP issue numbers from kubevirt/enhancements that are tracked in the target project
    # and are approved. A "Tracked" issue means that it is approved for the release.

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

    print(f"Fetching items from project 'kubevirt/projects/{project_number}' to check for 'Status: Tracked'")
    while has_next_page:
        variables["cursor"] = current_cursor
        result = execute_graphql_query(query, variables)

        if not result or "errors" in result:
            print(f"GraphQL query errors: {result.get('errors') if result else 'No response'}")
            break

        project_data = result.get("data", {}).get("organization", {}).get("projectV2", {})
        if not project_data or not project_data.get("items"):
            print(f"Warning: No items found for kubevirt project number {project_number}.")
            break
        
        items_data = project_data.get("items", {})
        nodes = items_data.get("nodes", [])
        
        for item_node in nodes:
            content = item_node.get("content")
            
            # Check if this is an Issue from the enhancements repo.
            if not (content and content.get("__typename") == "Issue" and
                    content.get("repository", {}).get("nameWithOwner") == "kubevirt/enhancements" and
                    "number" in content):
                continue

            # Check the status
            is_status_match = False
            for field_value_node in item_node.get("fieldValues", {}).get("nodes", []):
                if field_value_node.get("__typename") == "ProjectV2ItemFieldSingleSelectValue":
                    field = field_value_node.get("field", {})
                    if field.get("name") == "Status":
                        if field_value_node.get("name") == "Tracked":
                            is_status_match = True
                            break
            
            if is_status_match:
                tracked_issue_numbers.add(content["number"])
                print(f"  Found kubevirt/enhancements issue #{content['number']} with Status Tracked.")
            else:
                print(f"  Found kubevirt/enhancements issue #{content['number']} but status was not Tracked.")


        page_info = items_data.get("pageInfo", {})
        has_next_page = page_info.get("hasNextPage", False)
        current_cursor = page_info.get("endCursor") if has_next_page else None

    return tracked_issue_numbers

def main():
    # Get PR body
    pr_body = get_pr_details()
    if not pr_body:
        print("No PR body found.")
        return

    # Extract enhancements reference numbers
    ref_numbers = extract_enhancements_references(pr_body)
    if not ref_numbers:
        print("No enhancements references found.")
        return

    # Check the first reference
    ref_number = ref_numbers[0]
    # search for merged PRs mentioning the issue
    related_prs = find_related_merged_prs(ref_number)
    if related_prs:
        print(f"Found merged PR(s) for enhancements#{ref_number}: {related_prs}")
        add_label_to_pr()
    else:
        print(f"No merged PR found for enhancements#{ref_number}.")

if __name__ == "__main__":
    main()

