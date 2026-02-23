#!/bin/bash

gitlab-cli activity list --group-by-task --pipelines --json > ./../activity-cli/sources/gitlab.json
