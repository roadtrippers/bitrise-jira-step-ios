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

type user struct {
	Name  string
	Identifier string
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

	versionNumber := os.Getenv("version_number")
	if len(versionNumber) == 0 {
		fmt.Println("Error: No version number found!")
		os.Exit(1)
	}

	fmt.Printf("Using version number %v for Jira comments\n", versionNumber)

	buildNumber := os.Getenv("build_number")
	if len(buildNumber) == 0 {
		fmt.Println("Error: No build number found!")
		os.Exit(1)
	}

	fmt.Printf("Using build number %v for Jira comments\n", buildNumber)

	projectID := os.Getenv("jira_project_id")
	if len(projectID) == 0 {
		fmt.Println("Error: No Project ID found!")
		os.Exit(1)
	}

	fmt.Printf("Using Project ID: %v\n", projectID)

	branchID := os.Getenv("jira_branch_custom_field_id")
	if len(branchID) == 0 {
		fmt.Println("Error: No Branch ID found!")
		os.Exit(1)
	}

	fmt.Printf("Using Branch ID: %v\n", branchID)

	searchLabels := os.Getenv("jira_labels_to_search")
	if len(searchLabels) == 0 {
		fmt.Println("Error: No search labels found!")
		os.Exit(1)
	}

	fmt.Printf("Using Search Labels: %v\n", searchLabels)

	jiraLabelsToRemove := os.Getenv("jira_labels_to_remove")
	var removeLabelsJson []string
	if len(jiraLabelsToRemove) > 0 {
		jiraRemoveLabelsSlice := strings.Split(jiraLabelsToRemove, ",")
		for _, removeLabel := range jiraRemoveLabelsSlice {
			removeLabelsJson = append(removeLabelsJson, "{\"remove\": \"" + removeLabel + "\"}")
		}
		fmt.Printf("Labels to remove:%v\n", removeLabelsJson)
	} else {
		fmt.Println("No labels to remove found!")
	}

	jiraLabelsToAdd := os.Getenv("jira_labels_to_add")
	var addLabelsJson []string
	if len(jiraLabelsToAdd) > 0 {
		jiraAddLabelsSlice := strings.Split(jiraLabelsToAdd, ",")
		for _, addLabel := range jiraAddLabelsSlice {
			addLabelsJson = append(addLabelsJson, "{\"add\":\"" + addLabel + "\"}")
		}
		fmt.Printf("Labels to add: %v\n", addLabelsJson)
	} else {
		fmt.Println("No labels to add found!")
	}

	allLabels := append(addLabelsJson,removeLabelsJson...)
	allLabelsJson := ""
	if len(allLabels) > 0 {
		allLabelsJson = "\"labels\":[" + strings.Join(allLabels[:], ",") + "]"
	}

	bitriseBranch := os.Getenv("BITRISE_GIT_BRANCH")
	if len(bitriseBranch) > 0 {
		fmt.Printf("Using Branch: %v\n", bitriseBranch)
	} else {
		fmt.Println("No Branch found")
		os.Exit(1)
	}

	componentId := os.Getenv("jira_component_id")
	componentJson := ""
	if len(componentId) > 0 {
		componentJson = " AND component=\"" + componentId + "\""
		fmt.Printf("Component ID: %v\n", componentId)
	} else {
		fmt.Println("No Component ID found")
	}

	// Request Jira issues
	encodedParams := &url.URL{Path: "jql=project=" + projectID + " AND labels in (" + searchLabels + ") AND cf[" + branchID + "]~" + bitriseBranch + componentJson}
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

	jiraUsernames := os.Getenv("jira_username_list")
	mentionsJson := ""
	if len(jiraUsernames) > 0 {
		jiraUsernameSlice := strings.Split(jiraUsernames, ",")
		for _, usernameId := range jiraUsernameSlice {
			var usernameAndId []string
			usernameAndId = strings.Split(usernameId, ":")
			if len(usernameAndId) == 2 {
				mentionsJson = mentionsJson + "{\"type\":\"mention\",\"attrs\":{\"id\":\"" + usernameAndId[1] + "\",\"text\":\"" + usernameAndId[0] + "\",\"userType\":\"APP\"}},"
			}
		}
		fmt.Printf("Users to notify:%v\n", jiraUsernames)
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

			// make request to add and remove labels
			labelsURL := fmt.Sprintf("%s/rest/api/2/issue/%s", jiraURL, issue.Key)
			labelsJSONString := []byte(fmt.Sprintf("{\"update\":{%s}}", allLabelsJson))

			req, err = newRequest("PUT", labelsURL, bytes.NewBuffer(labelsJSONString))
			if err != nil {
				fmt.Printf("Error setting up jira labels request:%v\n", err)
				os.Exit(1)
			}

			resp, err = http.DefaultClient.Do(req)
			if err != nil {
				fmt.Printf("Error requesting jira labels update:%v\n", err)
				os.Exit(1)
			}
			defer resp.Body.Close()

			// make request to add comment and mentions
			commentsURL := fmt.Sprintf("%s/rest/api/3/issue/%s/comment", jiraURL, issue.Key)
			commentJSONString := []byte(fmt.Sprintf("{\"body\":{\"type\":\"doc\",\"version\":1,\"content\":[{\"type\":\"paragraph\",\"content\":[%s{\"text\":\" This will be in %s (%s)!\",\"type\": \"text\"}]}]}}", mentionsJson, versionNumber, buildNumber))
			
			req, err = newRequest("POST", commentsURL, bytes.NewBuffer(commentJSONString))
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
