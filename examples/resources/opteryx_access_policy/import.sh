#!/bin/sh
# The import ID is "workspace/policy_id" -- a policy ID alone doesn't say
# which workspace's Firestore subcollection it lives in.
terraform import opteryx_access_policy.analytics_reader analytics/3f2b1a9e-1234-4c56-9abc-0123456789ab
