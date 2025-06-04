import os
import re
import requests

GITHUB_TOKEN = os.environ["GITHUB_TOKEN"]
KUBEVIRT_REPO = os.environ.get("GITHUB_REPOSITORY", "kubevirt/kubevirt")
PR_NUMBER = os.environ["PR_NUMBER"]
HEADERS = {"Authorization": f"token {GITHUB_TOKEN}", "Accept": "application/vnd.github.v3+json"}

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

    # Extract the first non-empty group from each match
    ref_numbers = [group[0] or group[1] for group in matches]
    # there can be duplicates
    return list(set(ref_numbers))

def find_related_merged_prs(issue_number):
    #Find merged PRs in kubevirt/enhancements that reference a given issue.
    
    # Construct the search query for merged PRs referencing the issue
    query = (
        f'repo:kubevirt/enhancements is:pr is:merged '
    )
    base_url = "https://api.github.com/search/issues"
    params = {"q": query, "per_page": 100}
    
    related_prs = []
    page = 1
    
    while True:
        try:
            # Fetch the current page of results
            response = requests.get(
                f"{base_url}?q={params['q']}&per_page={params['per_page']}&page={page}",
                headers=HEADERS
            )
            if response.status_code != 200:
                print(f"API error: {response.status_code} - {response.text}")
                break
            
            data = response.json()
            items = data.get("items", [])
            
            # Extract PR numbers from the results
            for item in items:
                pr = item.get('pull_request', {})
                # Check if PR is merged and references the issue in its body
                # that's a bit of a mess since there can be multiple ways to reference 
                body = item.get('body', '')
                if (pr['merged_at'] and
                    (f"#{issue_number}" in body or
                     f"issues/{issue_number}" in body or
                     f"https://github.com/kubevirt/enhancements/issues/{issue_number}" in body)):
                    related_prs.append(item["number"])
            
            # If fewer than 100 items, we've reached the last page
            if len(items) < 100:
                break
            
            page += 1
        
        except Exception as e:
            print(f"Request failed: {e}")
            break
    
    return related_prs

def add_label_to_pr():
    # Adds the 'approved-vep' label to the kubevirt PR. (when we will have that )
    url = f"https://api.github.com/repos/{KUBEVIRT_REPO}/issues/{PR_NUMBER}/labels"
    payload = {"labels": ["approved-vep"]}
    response = requests.post(url, headers=HEADERS, json=payload)
    response.raise_for_status()

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

