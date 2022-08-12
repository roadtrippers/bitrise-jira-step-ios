| :warning: This repo is public: Do not commit any sensitive data. |
| --- |

# Jira Management & Release Notes

Creates release notes and stores them in an environment variable.  Updates issues with comments and transitions them to QA.


## How to use this Step

This will need to be added to your bitrise.yml file in `direct git clone` fashion.

Ex. 
```
- git::https://github.com/roadtrippers/bitrise-jira-step-ios.git@master:
    title: Jira Release Notes & Comments
```

### Required Parameters:
- **Jira Username:** This is the Jira username used for API authorization.
- **Jira API Token:** This is the token used API authorization. Can be found in the Jira account settings under Security.
- **Jira Organization URL:** The organization URL prefaced with https://.
- **Version Number:** The version number used for release notes.
- **Build Number:** The build number used for release notes.
- **Project ID:** The project ID used to filter issues.
- **Branch field ID:** This is the ID of the Branch field in Jira.  This is used to filter all issues to the github branch of the current build.
- **Labels to filter by:** A list of labels used to filter issues.  These need to be comma seperated.
- **Labels to remove:** A list of labels to remove from transitioned issues.
- **Labels to add:** A list of labels to add to transitioned issues.
- **Jira Usernames:** A list of Display Name and User Id pairs.  They need to be in the format Name:ID seprated by commas.
- **Jira Transition ID:** The ID used to transition an issue from state to another.

### Outputs:
- **$RELEASE_NOTES** Release notes in the form of a list of found issues with the version and build numbers as well as the commit string.

