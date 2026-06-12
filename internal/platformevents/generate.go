package platformevents

// Regenerates the deploy.created Go bindings from the EventBridge Schema
// Registry. Requires AWS credentials with schema registry read access; skips
// when CI=true
//go:generate go tool github.com/Clever/slingshot/cmd/schemabindings -registry arn:aws:schemas:us-west-2:605134456190:registry/clever-events -schema deploy.created -out ./schemas/deploycreated/deploycreated.gen.go -region us-west-2
