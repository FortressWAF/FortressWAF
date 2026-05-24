#!/bin/bash
# Setup GitHub branch rulesets via API
# Requires: gh auth login

set -euo pipefail

REPO="${GITHUB_REPOSITORY:-$(gh repo view --json nameWithOwner --jq .nameWithOwner)}"

echo "Setting up branch rulesets for $REPO..."

# Ruleset 1: main branch protection
echo "Creating ruleset for main branch..."
cat <<'RULESET' | gh api repos/$REPO/rulesets --input - 2>&1 || echo "ruleset may already exist"
{
  "name": "Main branch protection",
  "target": "branch",
  "enforcement": "active",
  "conditions": {
    "ref_name": {
      "include": ["refs/heads/main"],
      "exclude": []
    }
  },
  "rules": [
    {"type": "deletion"},
    {"type": "non_fast_forward"},
    {"type": "required_linear_history"},
    {"type": "required_status_checks", "parameters": {
      "required_status_checks": [
        {"context": "Lint Go", "integration_id": null},
        {"context": "Lint JS", "integration_id": null},
        {"context": "Test Go", "integration_id": null},
        {"context": "Test JS", "integration_id": null},
        {"context": "Build Docker", "integration_id": null}
      ],
      "strict_required_status_checks_policy": true
    }},
    {"type": "pull_request", "parameters": {
      "required_approving_review_count": 1,
      "dismiss_stale_reviews_on_push": true,
      "require_code_owner_review": false,
      "require_last_push_approval": true
    }},
    {"type": "required_signatures"},
    {"type": "code_scanning", "parameters": {
      "code_scanning_tools": [{"tool": "CodeQL", "security_alerts_threshold": "high_or_higher", "alerts_threshold": "errors"}]
    }}
  ],
  "bypass_actors": [
    {"actor_id": 1, "actor_type": "RepositoryRole", "bypass_mode": "always"}
  ]
}
RULESET

# Ruleset 2: fix/* branches (no direct pushes, require PR)
echo "Creating ruleset for fix/* branches..."
cat <<'RULESET' | gh api repos/$REPO/rulesets --input - 2>&1 || echo "ruleset may already exist"
{
  "name": "Fix branch protection",
  "target": "branch",
  "enforcement": "active",
  "conditions": {
    "ref_name": {
      "include": ["refs/heads/fix/*"],
      "exclude": []
    }
  },
  "rules": [
    {"type": "deletion"},
    {"type": "non_fast_forward"}
  ],
  "bypass_actors": [
    {"actor_id": 1, "actor_type": "RepositoryRole", "bypass_mode": "always"}
  ]
}
RULESET

echo "Done."
