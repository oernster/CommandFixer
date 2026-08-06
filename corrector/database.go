package corrector

// The data the engine matches against, kept apart from the matching itself.
//
// This is the half of the package that grows: every new tool, subcommand or
// Windows built-in lands here and none of it changes how correction works.
// Splitting it out means adding a tool never touches the correction logic and
// never pushes that file towards the size cap.

import "sort"

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

	// PowerShell POSIX-style aliases. These are valid commands in the target
	// shell, so they must be recognised exactly and never "corrected" (for
	// example "ls" is one insertion from "cls" and would otherwise be rewritten
	// to it). Get-ChildItem, Get-Content, Copy-Item, etc.
	"ls", "cat", "cp", "mv", "rm", "rmdir", "pwd", "ps", "clear",
	"kill", "man", "tee", "diff", "curl", "wget", "sleep", "history",
}

// commandAliases maps habitual command-name typos to their intended command.
// These are corrected unconditionally (independent of the similarity threshold)
// because they are transpositions a user makes every time, for example "gti"
// for "git". Add further always-wrong spellings here.
var commandAliases = map[string]string{
	"gti": "git",
}

// commandDBTools is the sorted list of known CLI tool names (the keys of
// commandDB), used to fuzzy-correct a mistyped tool name such as "dokcer" for
// "docker". It is sorted so that correction is deterministic.
var commandDBTools = sortedToolNames()

// sortedToolNames returns the commandDB keys in sorted order.
func sortedToolNames() []string {
	names := make([]string, 0, len(commandDB))
	for name := range commandDB {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
