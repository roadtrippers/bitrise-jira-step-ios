package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/tidwall/gjson"
)

type issue struct {
	Key  string
	Name string
}

func newRequest(method, url string, body io.Reader) (*http.Request, error) {

	username := os.Getenv("jira_username")
	if len(username) == 0 {
		fmt.Println("Error: No username found!")
		os.Exit(1)
	}

	token := os.Getenv("jira_api_token_string")
	if len(token) == 0 {
		fmt.Println("Error: No token found!")
		os.Exit(1)
	}

	req, err := http.NewRequest(method, url, body)
	req.SetBasicAuth(username, token)
	req.Header.Set("Content-Type", "application/json")

	return req, err
}

func main() {

	jiraURL := os.Getenv("jira_organization_url")
	if len(jiraURL) == 0 {
		fmt.Println("Error: No organization URL found!")
		os.Exit(1)
	}

	buildNumber := os.Getenv("build_number")
	if len(buildNumber) == 0 {
		fmt.Println("Error: No build number found!")
		os.Exit(1)
	}

	fmt.Printf("Using build number %v for Jira comments\n", buildNumber)

	branchID := os.Getenv("jira_branch_custom_field_id")
	if len(branchID) == 0 {
		fmt.Println("Error: No branch ID found!")
		os.Exit(1)
	}

	fmt.Printf("Using Branch ID: %v\n", branchID)

	needsID := os.Getenv("jira_needs_custom_field_id")
	needsJSON := ""
	if len(needsID) > 0 {
		fmt.Printf("Using Needs ID: %v\n", needsID)
		needsJSON = ",\"customfield_" + needsID + "\":[{\"remove\":{\"value\":\"Build\"}}]"
	} else {
		fmt.Println("No custom field Needs ID found, updating Needs field.")
		os.Exit(1)
	}

	// Request Jira issues
	encodedParams := &url.URL{Path: "jql=project=RTIOS AND cf[" + needsID + "]=Build AND cf[" + branchID + "]~" + os.Getenv("BITRISE_GIT_BRANCH")}
	encodedString := encodedParams.String()
	encodedURL := jiraURL + "/rest/api/3/search?" + encodedString
	req, err := newRequest("GET", encodedURL, nil)
	if err != nil {
		fmt.Printf("Error setting up jira issue request:%v\n", err)
		os.Exit(1)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error requesting Jira issues %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	// Create issue structs based on whether issue branch equals workflow branch
	var issues []issue
	allIssues := gjson.Get(string(body), "issues")

	for _, result := range allIssues.Array() {
		branch := result.Get("fields.customfield_" + branchID)
		if branch.String() == os.Getenv("BITRISE_GIT_BRANCH") {
			issue := issue{result.Get("key").String(), result.Get("fields.summary").String()}
			issues = append(issues, issue)
		}
	}

	// Construct release notes
	var buf bytes.Buffer
	for _, issue := range issues {
		buf.WriteString(fmt.Sprintf("%s", issue.Name))
		buf.WriteString("\n")
	}

	buf.WriteString(fmt.Sprintf("\n%s - %s", os.Getenv("BITRISE_GIT_BRANCH"), os.Getenv("GIT_CLONE_COMMIT_HASH")))
	releaseNotes := buf.String()

	// Create environment variable for release notes
	c := exec.Command("envman", "add", "--key", "RELEASE_NOTES", "--value", releaseNotes)
	err = c.Run()
	if err != nil {
		fmt.Printf("Error setting RELEASE_NOTES environment variable:%v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Release Notes Created:%v\n", releaseNotes)

	jiraUsernames := strings.Replace(os.Getenv("jira_username_list"), " ", "", -1)
	usernameTags := ""
	if len(jiraUsernames) > 0 {
		jiraUsernameSlice := strings.Split(jiraUsernames, ",")
		for _, username := range jiraUsernameSlice {
			usernameTags = usernameTags + "[~" + username + "]"
		}
		fmt.Printf("Usernames to notify:%v\n", usernameTags)
	} else {
		fmt.Println("No usernames found, not notifying jira users.")
	}

	transitionID := os.Getenv("jira_transition_id")
	transitionJSON := ""
	if len(transitionID) > 0 {
		fmt.Printf("Transition ID to use:%v\n", transitionID)
		transitionJSON = `{"transition":{"id":"` + transitionID + `"}}`
	} else {
		fmt.Println("No transition ID found, not transitioning issues.")
	}

	if len(issues) > 0 {
		fmt.Printf("Issues found:%v\n", issues)
		// Parse json issue keys
		for _, issue := range issues {
			if len(transitionJSON) > 0 {
				// make transition request
				transitionURL := fmt.Sprintf("%s/rest/api/3/issue/%s/transitions", jiraURL, issue.Key)
				transitionJSONString := []byte(transitionJSON)
				req, err = newRequest("POST", transitionURL, bytes.NewBuffer(transitionJSONString))
				if err != nil {
					fmt.Printf("Error setting up jira transition request:%v\n", err)
					os.Exit(1)
				}

				resp, err = http.DefaultClient.Do(req)
				if err != nil {
					fmt.Printf("Error requesting jira transition:%v\n", err)
					os.Exit(1)
				}
				defer resp.Body.Close()
			}

			// make request to add comment and remove needs build
			commentsURL := fmt.Sprintf("%s/rest/api/2/issue/%s", jiraURL, issue.Key)
			commentJSONString := []byte(fmt.Sprintf("{\"update\":{\"comment\":[{\"add\":{\"body\":\"%s This will be in build %s!\"}}]%s}}", usernameTags, buildNumber, needsJSON))
			req, err = newRequest("PUT", commentsURL, bytes.NewBuffer(commentJSONString))
			if err != nil {
				fmt.Printf("Error setting up jira comment request:%v\n", err)
				os.Exit(1)
			}

			resp, err = http.DefaultClient.Do(req)
			if err != nil {
				fmt.Printf("Error requesting jira comments:%v\n", err)
				os.Exit(1)
			}
			defer resp.Body.Close()
		}
	} else {
		fmt.Println("No issues found!")
	}

	os.Exit(0)
}
