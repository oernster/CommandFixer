// Package corrector applies fuzzy-matching typo correction to shell commands.
//
// It maintains a built-in database of popular CLI tools and their valid
// subcommands. When a typed command looks like a typo of a known subcommand
// (within a configurable similarity threshold), Suggest returns the corrected
// form. No user-maintained dictionary is required.
package corrector

import "strings"

// commandDB maps known CLI tool names to their valid subcommands.
// Tool names must match exactly; subcommands are matched by similarity.
var commandDB = map[string][]string{
	"git": {
		"add", "bisect", "blame", "branch", "checkout", "cherry-pick",
		"clean", "clone", "commit", "describe", "diff", "fetch",
		"format-patch", "grep", "init", "log", "merge", "mv",
		"pull", "push", "rebase", "remote", "reset", "revert",
		"rm", "show", "stash", "status", "submodule", "switch", "tag",
	},
	"docker": {
		"build", "commit", "compose", "container", "cp", "exec",
		"image", "images", "info", "inspect", "kill", "login",
		"logout", "logs", "network", "ps", "pull", "push",
		"rm", "rmi", "run", "start", "stats", "stop",
		"system", "tag", "top", "volume",
	},
	"kubectl": {
		"annotate", "api-resources", "api-versions", "apply", "attach",
		"auth", "autoscale", "certificate", "cluster-info", "completion",
		"config", "cordon", "cp", "create", "delete", "describe",
		"drain", "edit", "exec", "explain", "expose", "get",
		"label", "logs", "patch", "port-forward", "proxy", "replace",
		"rollout", "run", "scale", "set", "taint", "top",
		"uncordon", "version", "wait",
	},
	"npm": {
		"audit", "build", "cache", "ci", "exec", "fund",
		"help", "init", "install", "link", "list", "outdated",
		"pack", "ping", "publish", "rebuild", "restart", "root",
		"run", "start", "stop", "test", "uninstall", "update",
		"version", "view",
	},
	"yarn": {
		"add", "audit", "build", "cache", "check", "config",
		"create", "exec", "global", "help", "import", "info",
		"init", "install", "link", "list", "login", "logout",
		"outdated", "owner", "pack", "policies", "publish", "remove",
		"run", "start", "tag", "team", "test", "unlink",
		"upgrade", "version", "versions", "why", "workspace", "workspaces",
	},
	"cargo": {
		"add", "bench", "build", "check", "clean", "clippy",
		"doc", "fetch", "fix", "fmt", "help", "init",
		"install", "login", "logout", "metadata", "new", "owner",
		"package", "publish", "remove", "report", "run", "rustc",
		"rustdoc", "search", "test", "tree", "uninstall", "update",
		"vendor", "version", "yank",
	},
	"go": {
		"build", "clean", "doc", "env", "fix", "fmt",
		"generate", "get", "help", "install", "list", "mod",
		"run", "telemetry", "test", "tool", "version", "vet", "work",
	},
	"pip": {
		"cache", "check", "completion", "config", "debug", "download",
		"freeze", "hash", "help", "index", "inspect", "install",
		"list", "search", "show", "uninstall", "wheel",
	},
	"pip3": {
		"cache", "check", "completion", "config", "debug", "download",
		"freeze", "hash", "help", "index", "inspect", "install",
		"list", "search", "show", "uninstall", "wheel",
	},
	"terraform": {
		"apply", "console", "destroy", "fmt", "force-unlock",
		"get", "graph", "import", "init", "login", "logout",
		"metadata", "output", "plan", "providers", "refresh",
		"show", "state", "taint", "test", "untaint", "validate",
		"version", "workspace",
	},
	"helm": {
		"completion", "create", "dependency", "env", "get", "help",
		"history", "install", "lint", "list", "package", "plugin",
		"pull", "push", "registry", "repo", "rollback", "search",
		"show", "status", "template", "test", "uninstall", "upgrade",
		"verify", "version",
	},
	"az": {
		"account", "acr", "aks", "apim", "appservice", "backup",
		"batch", "bicep", "billing", "bot", "cache", "cdn",
		"cloud", "cognitiveservices", "config", "configure", "container",
		"cosmosdb", "deployment", "devops", "disk", "dns",
		"eventgrid", "eventhub", "extension", "feature", "find",
		"functionapp", "group", "identity", "image", "iot",
		"keyvault", "lock", "login", "logout", "monitor",
		"mysql", "network", "policy", "postgres", "redis",
		"resource", "role", "search", "security", "servicebus",
		"snapshot", "sql", "ssh", "storage", "tag",
		"upgrade", "version", "vm", "vmss", "webapp",
	},
	"aws": {
		"acm", "apigateway", "batch", "cloudformation", "cloudfront",
		"cloudtrail", "cloudwatch", "codebuild", "codecommit", "codedeploy",
		"codepipeline", "configure", "dynamodb", "ec2", "ecr",
		"ecs", "eks", "elasticache", "elasticbeanstalk", "elbv2",
		"emr", "iam", "kinesis", "kms", "lambda", "lightsail",
		"logs", "organizations", "rds", "redshift", "route53",
		"s3", "s3api", "secretsmanager", "ses", "sns",
		"sqs", "ssm", "stepfunctions", "sts", "xray",
	},
	"gcloud": {
		"alpha", "app", "artifacts", "auth", "beta", "bigtable",
		"builds", "components", "composer", "compute", "config",
		"container", "dataflow", "dataproc", "datastore", "deploy",
		"dns", "domains", "filestore", "firestore", "functions",
		"help", "iam", "info", "init", "kms", "logging",
		"monitoring", "organizations", "projects", "pubsub", "redis",
		"run", "scheduler", "secrets", "services", "source",
		"spanner", "sql", "storage", "tasks", "version", "workflows",
	},
}

// defaultThreshold is used when New receives a zero or out-of-range threshold.
const defaultThreshold = 0.6

// Engine performs fuzzy subcommand correction for known CLI tools.
type Engine struct {
	threshold float64
}

// New creates an Engine with the given similarity threshold (0.0, 1.0].
// A zero or out-of-range value applies the package default (0.6).
func New(threshold float64) *Engine {
	if threshold <= 0 || threshold > 1 {
		threshold = defaultThreshold
	}
	return &Engine{threshold: threshold}
}

// Threshold returns the similarity threshold in use.
func (e *Engine) Threshold() float64 {
	return e.threshold
}

// Suggest checks whether cmd contains a recognisable typo in its subcommand
// position. It returns the corrected form and true when a similar-enough match
// is found in the built-in command database; otherwise it returns cmd unchanged
// and false.
//
// Rules:
//   - The tool name (first token) must be an exact key in commandDB.
//   - At least two tokens must be present.
//   - If the subcommand (second token) is already a known valid subcommand,
//     no correction is made.
//   - All tokens beyond the second are preserved verbatim.
func (e *Engine) Suggest(cmd string) (string, bool) {
	tokens := strings.Fields(cmd)
	if len(tokens) < 2 {
		return cmd, false
	}

	tool := tokens[0]
	subcommand := tokens[1]

	subcommands, known := commandDB[tool]
	if !known {
		return cmd, false
	}

	// Already a valid subcommand: nothing to correct.
	for _, sc := range subcommands {
		if sc == subcommand {
			return cmd, false
		}
	}

	// Find the closest subcommand by normalised Levenshtein similarity.
	bestMatch := ""
	bestSim := 0.0
	for _, sc := range subcommands {
		sim := similarity(subcommand, sc)
		if sim > bestSim {
			bestSim = sim
			bestMatch = sc
		}
	}

	if bestSim < e.threshold || bestMatch == "" {
		return cmd, false
	}

	tokens[1] = bestMatch
	return strings.Join(tokens, " "), true
}

// similarity returns a value in [0, 1] representing how alike a and b are,
// based on normalised Levenshtein distance over byte length.
func similarity(a, b string) float64 {
	maxLen := max(len(a), len(b))
	if maxLen == 0 {
		return 1.0
	}
	dist := levenshtein(a, b)
	return 1.0 - float64(dist)/float64(maxLen)
}

// levenshtein computes the edit distance between two ASCII strings
// using a two-row dynamic programming approach.
func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, min(prev[j]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}
	return prev[lb]
}
