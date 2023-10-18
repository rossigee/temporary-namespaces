# Temporary namespaces

This microservice is designed to run with a role that allows it to query the annotations on all namespace, and delete any that have a specific 'expiry' timestamp annotation that has since expired.

This allows us to spin up 'temporary namespaces' from our CI/CD build system, run a series of automated tests against them and announce them to the testing team who can perform any manual tests as required.

## Configuration

Start with a little [RBAC](examples/rbac.yaml).

Then, a [deployment](examples/deployment.yaml).

Put this in your flux2/ArgoCD repo :)
