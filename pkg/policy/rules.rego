package scim.authz

# Use the universal v1 compatibility syntax
import rego.v1

# Default decision is strict deny
default allow = false

# Allow decision logic
# Added the 'if' keyword before the conditional body
allow if {
	# 1. Agent must be active
	input.agent.active == true

	# 2. Human Owner must be active
	input.owner.active == true

	# 3. Execution action must be explicitly allowed for the agent
	input.action in input.agent.scopes

	# 4. Owner must possess the required enterprise group/role for the target tool
	user_has_required_group
}

# Helper rule: Validate group membership (ReBAC)
# Added the 'if' keyword here as well
user_has_required_group if {
	some group in input.owner.groups
	group == input.target_required_group
}
