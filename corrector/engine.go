// Package corrector applies fuzzy-matching typo correction to shell commands.
//
// It maintains a built-in database of popular CLI tools and their valid
// subcommands. When a typed command looks like a typo of a known subcommand
// (within a configurable similarity threshold), Suggest returns the corrected
// form. No user-maintained dictionary is required.
//
// For Windows standalone commands (dir, cd, copy, etc.), Suggest also
// fuzzy-matches the first token against a known standalone command list
// and corrects the command name itself when a close-enough match is found.
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

	// Windows package managers and CLI tools with subcommand structure.
	"winget": {
		"configure", "download", "export", "features", "hash",
		"import", "install", "list", "pin", "search", "settings",
		"show", "source", "uninstall", "upgrade", "validate",
	},
	"choco": {
		"apikey", "config", "export", "feature", "find", "help",
		"info", "install", "list", "new", "optimize", "outdated",
		"pack", "pin", "push", "search", "setapikey", "source",
		"sources", "sync", "template", "uninstall", "unpackself",
		"upgrade", "version",
	},
	"scoop": {
		"alias", "bucket", "cache", "cat", "checkup", "cleanup",
		"config", "create", "depends", "download", "export", "help",
		"hold", "home", "import", "info", "install", "list",
		"prefix", "reset", "search", "shim", "status",
		"unhold", "uninstall", "update", "utils", "virustotal",
	},

	// Windows built-in admin tools with subcommand structure.
	"net": {
		"accounts", "computer", "config", "continue", "file",
		"group", "help", "helpmsg", "localgroup", "pause",
		"print", "send", "session", "share", "start",
		"statistics", "stop", "time", "use", "user", "view",
	},
	"sc": {
		"boot", "config", "continue", "control", "create",
		"delete", "description", "failure", "failureflag",
		"lock", "pause", "qc", "qdescription", "qfailure",
		"qfailureflag", "query", "queryex", "querylock",
		"sdset", "sdshow", "showsid", "sidtype",
		"start", "stop", "triggerinfo",
	},
	"reg": {
		"add", "compare", "copy", "delete", "export",
		"flags", "import", "load", "query", "restore",
		"save", "unload",
	},
	"netsh": {
		"advfirewall", "branchcache", "bridge", "dhcpclient",
		"dnsclient", "firewall", "http", "interface",
		"ipsec", "lan", "namespace", "netio",
		"ras", "rpc", "trace", "wfp",
		"winhttp", "winsock", "wlan",
	},
}

// windowsCommands is the list of known Windows standalone commands.
// When the first token of a command fuzzy-matches one of these entries
// and the tool is not already a key in commandDB, Suggest corrects the
// command name (first token) and preserves all remaining tokens verbatim.
var windowsCommands = []string{
	// Navigation
	"cd", "chdir", "pushd", "popd",
	// File operations
	"attrib", "cipher", "compact", "copy", "del", "erase", "fc",
	"find", "findstr", "fsutil", "icacls", "mklink", "move",
	"recover", "ren", "rename", "replace", "robocopy", "xcopy",
	// Directory operations
	"dir", "md", "mkdir", "rd", "rmdir", "tree",
	// Display / text output
	"cls", "color", "echo", "more", "sort", "type",
	// Disk and filesystem
	"chkdsk", "diskpart", "format", "label", "subst",
	// System information and diagnostics
	"driverquery", "hostname", "ipconfig", "netstat", "nslookup",
	"ping", "systeminfo", "tasklist", "tracert", "ver",
	"where", "whoami",
	// Process management
	"start", "taskkill", "timeout",
	// Configuration and policy
	"bcdedit", "gpupdate", "mode", "msiexec", "path", "prompt",
	"set", "setx", "sfc", "shutdown", "title",
	// Miscellaneous
	"assoc", "date", "msg", "pause", "print", "schtasks", "time",
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

// Suggest checks whether cmd contains a recognisable typo and returns the
// corrected form and true when a similar-enough match is found. It returns
// cmd unchanged and false when no correction can be made.
//
// Two correction modes are applied in order:
//
//  1. Subcommand correction: when the first token (tool name) is an exact key
//     in commandDB, the second token (subcommand) is fuzzy-matched against
//     the tool's known subcommands. All tokens beyond the second are preserved.
//
//  2. Standalone command correction: when the first token is not in commandDB,
//     it is fuzzy-matched against the windowsCommands list. When a close-enough
//     match is found the command name is corrected and all remaining tokens are
//     preserved verbatim.
//
// In both modes at least two tokens must be present and the similarity must
// meet or exceed the configured threshold.
func (e *Engine) Suggest(cmd string) (string, bool) {
	tokens := strings.Fields(cmd)
	if len(tokens) < 2 {
		return cmd, false
	}

	tool := tokens[0]
	subcommand := tokens[1]

	subcommands, known := commandDB[tool]
	if !known {
		return e.suggestStandalone(cmd, tokens)
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

// suggestStandalone attempts to correct the first token of tokens by
// fuzzy-matching it against the windowsCommands list. It returns the corrected
// command and true when a match at or above the threshold is found, or the
// original cmd and false otherwise.
func (e *Engine) suggestStandalone(cmd string, tokens []string) (string, bool) {
	tool := tokens[0]

	// Already an exact known standalone command: no correction needed.
	for _, sc := range windowsCommands {
		if sc == tool {
			return cmd, false
		}
	}

	bestMatch := ""
	bestSim := 0.0
	for _, sc := range windowsCommands {
		sim := similarity(tool, sc)
		if sim > bestSim {
			bestSim = sim
			bestMatch = sc
		}
	}

	if bestSim < e.threshold || bestMatch == "" {
		return cmd, false
	}

	tokens[0] = bestMatch
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
