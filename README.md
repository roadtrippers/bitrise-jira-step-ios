| :warning: This repo is public: Do not commit any sensitive data. |
| --- |

# Jira Management & Release Notes

Creates release notes and stores them in an environment variable.  Updates issues with comments and transitions them to QA.


## How to use this Step

This will need to be added to you bitrise.yml file in direct git clone fashion.

Ex. 
```
- git::https://github.com/roadtrippers/bitrise-jira-step-ios.git@master:
    title: Jira Release Notes & Comments
```

### Required Parameters:
- **Jira Username:** This is the Jira username used for API authorization.
- **Jira API Token:** This is the token used API authorization. Can be found in the Jira account settings under Security.
- **Build Number:** The build number used for release notes.
- **Custom Field ID for Branch:** This is the ID of the Branch field in Jira.  This is used to filter all issues to the github branch of the current build.
- **Custom Field ID for Needs:** This is the ID of the Needs field in Jira.  This is used to filter all issues to only those of a specific need.