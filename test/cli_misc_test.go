package test

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/fnproject/cli/testharness"
)

func TestFnVersion(t *testing.T) {
	t.Parallel()

	tctx := testharness.Create(t)
	res := tctx.Fn("version")
	res.AssertSuccess()
}

func withMinimalFunction(h *testharness.CLIHarness) {

	h.CopyFiles(map[string]string{
		"simplefunc/vendor":    "vendor",
		"simplefunc/func.go":   "func.go",
		"simplefunc/go.sum":    "go.sum",
		"simplefunc/go.mod":    "go.mod",
		"simplefunc/func.yaml": "func.yaml",
	})
}

func withMinimalOCIApplication(h *testharness.CLIHarness) {
	if h.IsOCITestMode() {
		h.CopyFiles(map[string]string{
			"simpleapp/app.json":   "app.json",
		})

		h.CopyFilesToHomeDir(map[string]string{
			"oci-auth/fn/config.yaml": 						".fn/config.yaml",
			"oci-auth/fn/contexts/functions-test.yaml":    	".fn/contexts/functions-test.yaml",
			"oci-auth/oci/config":   						".oci/config",
			"docker":   									".docker",
		})
	}
}

// this is messy and nasty  as we generate different potential values for FN_API_URL based on its type
func fnApiUrlVariations(t *testing.T) []string {

	srcUrl := os.Getenv("FN_API_URL")

	if srcUrl == "" {
		srcUrl = "http://localhost:8080/"
	}

	if !strings.HasPrefix(srcUrl, "http:") && !strings.HasPrefix(srcUrl, "https:") {
		srcUrl = "http://" + srcUrl
	}
	parsed, err := url.Parse(srcUrl)

	if err != nil {
		t.Fatalf("Invalid/unparsable TEST_API_URL %s: %s", srcUrl, err)
	}

	var cases []string

	if parsed.Scheme == "http" {
		cases = append(cases, "http://"+parsed.Host+parsed.Path)
		cases = append(cases, parsed.Host+parsed.Path)
		cases = append(cases, parsed.Host)
	} else if parsed.Scheme == "https" {
		cases = append(cases, "https://"+parsed.Host+parsed.Path)
		cases = append(cases, "https://"+parsed.Host)
	} else {
		log.Fatalf("Unsupported url scheme for testing %s: %s", srcUrl, parsed.Scheme)
	}

	return cases
}

func TestFnApiUrlSupportsDifferentFormats(t *testing.T) {
	t.Parallel()

	h := testharness.Create(t)
	defer h.Cleanup()

	withMinimalOCIApplication(h)
	for _, candidateUrl := range fnApiUrlVariations(t) {
		h.WithEnv("FN_API_URL", candidateUrl)
		h.Fn("list", "apps").AssertSuccess()
	}
}

// Not sure what this test was intending (copied from old test.sh)
func TestSettingTimeoutWorks(t *testing.T) {
	t.Parallel()

	h := testharness.Create(t)
	defer h.Cleanup()
	h.WithEnv("FN_REGISTRY", h.GetFnRegistry())

	appName := h.NewAppName()
	withMinimalOCIApplication(h)
	h.Fn("create", "app", appName, "--annotation", h.GetSubnetAnnotation()).AssertSuccess()
	res := h.Fn("list", "apps")

	if !strings.Contains(res.Stdout, fmt.Sprintf("%s", appName)) {
		t.Fatalf("Expecting list apps to contain app name , got %v", res)
	}

	funcName := h.NewFuncName(appName)

	h.MkDir(funcName)
	h.Cd(funcName)
	withMinimalFunction(h)
	withMinimalOCIApplication(h)
	h.FileAppend("func.yaml", "\ntimeout: 50\n\nschema_version: 20180708\n")
	h.Fn("deploy", "--app", appName).AssertSuccess()
	h.Fn("invoke", appName, funcName).AssertSuccess()

	inspectRes := h.Fn("inspect", "fn", appName, funcName)
	inspectRes.AssertSuccess()
	if !strings.Contains(inspectRes.Stdout, `"timeout": 50`) {
		t.Errorf("Expecting fn inspect to contain CPU %v", inspectRes)
	}

	anotherFuncName := h.NewFuncName(appName)
	h.Fn("create", "fn", appName, anotherFuncName, h.GetFnImage()).AssertSuccess()

	h.Fn("invoke", appName, anotherFuncName).AssertSuccess()
}

//Memory doesn't seem to get persisted/returned
func TestSettingMemoryWorks(t *testing.T) {
	t.Parallel()

	h := testharness.Create(t)
	defer h.Cleanup()
	h.WithEnv("FN_REGISTRY", h.GetFnRegistry())

	appName := h.NewAppName()
	withMinimalOCIApplication(h)
	h.Fn("create", "app", appName, "--annotation", h.GetSubnetAnnotation()).AssertSuccess()
	res := h.Fn("list", "apps")

	if !strings.Contains(res.Stdout, fmt.Sprintf("%s", appName)) {
		t.Fatalf("Expecting list apps to contain app name , got %v", res)
	}

	funcName := h.NewFuncName(appName)

	h.MkDir(funcName)
	h.Cd(funcName)
	withMinimalFunction(h)
	h.FileAppend("func.yaml", "memory: 128\nschema_version: 20180708\n")
	h.Fn("deploy", "--app", appName).AssertSuccess()
	h.Fn("invoke", appName, funcName).AssertSuccess()

	inspectRes := h.Fn("inspect", "fn", appName, funcName)
	inspectRes.AssertSuccess()
	if !strings.Contains(inspectRes.Stdout, `"memory": 128`) {
		t.Errorf("Expecting fn inspect to contain CPU %v", inspectRes)
	}

	anotherFuncName := h.NewFuncName(appName)
	h.Fn("create", "fn", appName, anotherFuncName, h.GetFnImage()).AssertSuccess()

	h.Fn("invoke", appName, anotherFuncName).AssertSuccess()
}

func TestAllMainCommandsExist(t *testing.T) {
	t.Parallel()

	h := testharness.Create(t)
	defer h.Cleanup()

	testCommands := []string{
		"build",
		"bump",
		"call",
		"create",
		"delete",
		"deploy",
		"get",
		"init",
		"inspect",
		"list",
		"push",
		"run",
		"set",
		"test",
		"unset",
		"update",
		"use",
		"version",
	}

	if !h.IsOCITestMode() {
		// start command refers to starting local fn server. Not Applicable to OCI test mode.
		testCommands = append(testCommands, "start")
	}

	for _, cmd := range testCommands {
		res := h.Fn(cmd)
		if strings.Contains(res.Stderr, "command not found") {
			t.Errorf("Expected command %s to exist", cmd)
		}
	}
}

func TestAppYamlDeployFailNotExist(t *testing.T) {
	t.Parallel()

	h := testharness.Create(t)
	defer h.Cleanup()

	appName := h.NewAppName()
	fnName := h.NewFuncName(appName)
	h.WithFile("app.yaml", fmt.Sprintf(`name: %s`, appName), 0644)
	h.MkDir(fnName)
	h.Cd(fnName)
	withMinimalFunction(h)
	withMinimalOCIApplication(h)
	h.Cd("")
	h.Fn("deploy", "--all").AssertFailed().AssertStderrContains("app " + appName + " not found")
}

func TestAppYamlDeployInspect(t *testing.T) {
	// this test only inspects, does not invoke! for syslog, mostly
	t.Parallel()

	h := testharness.Create(t)
	defer h.Cleanup()

	appName := h.NewAppName()
	fnName := h.NewFuncName(appName)
	h.WithFile("app.yaml", fmt.Sprintf(`
name: %s
syslog_url: tcp://example.com:42
annotations:
  oracle.com/oci/subnetIds:
  - %s
config:
  animal: giraffe
`, appName, h.GetSubnetID()), 0644)
	h.MkDir(fnName)
	h.Cd(fnName)
	withMinimalFunction(h)
	withMinimalOCIApplication(h)
	h.Cd("")
	h.Fn("deploy", "--all", "--create-app").AssertSuccess()
	// check config from app.yaml is set
	inspect := h.Fn("inspect", "app", appName).AssertSuccess()
	inspect.AssertStdoutContains(fmt.Sprintf(`"name": "%s"`, appName))
	inspect.AssertStdoutContains(`"giraffe"`)
	if !h.IsOCITestMode() {
		// No support for syslog_url in OCI test mode
		inspect.AssertStdoutContains(`"tcp://example.com:42"`)
	}

	// now should exist, this should work too
	h.WithFile("app.yaml", fmt.Sprintf(`
name: %s
syslog_url: tcp://example.com:443
annotations:
  oracle.com/oci/subnetIds:
  - %s
config:
  animal: giraffe
  tea: oolong
`, appName, h.GetSubnetID()), 0644)
	h.Fn("deploy", "--all").AssertSuccess()
	// make sure config was updated
	inspect = h.Fn("inspect", "app", appName).AssertSuccess()
	inspect.AssertStdoutContains(fmt.Sprintf(`"name": "%s"`, appName))
	inspect.AssertStdoutContains(`"oolong"`)
	if !h.IsOCITestMode() {
		// No support for syslog_url in OCI test mode
		inspect.AssertStdoutContains(`"tcp://example.com:443"`)
	}
}

func TestAppYamlDeploy(t *testing.T) {
	// this test makes sure that functions can be invoked after using app.yaml deploy,
	// the other test that looks just like this just inspects the function b/c syslog config
	// in tests is a bad idea (ie don't test syslog here)

	t.Parallel()

	h := testharness.Create(t)
	defer h.Cleanup()

	appName := h.NewAppName()
	fnName := h.NewFuncName(appName)
	h.WithFile("app.yaml", fmt.Sprintf(`
name: %s
annotations:
  oracle.com/oci/subnetIds:
  - %s
config:
  animal: giraffe
`, appName, h.GetSubnetID()), 0644)
	h.MkDir(fnName)
	h.Cd(fnName)
	withMinimalFunction(h)
	withMinimalOCIApplication(h)
	h.Cd("")
	h.Fn("deploy", "--all", "--create-app").AssertSuccess()
	h.Fn("invoke", appName, fnName).AssertSuccess()
	// check config from app.yaml is set
	h.Fn("get", "config", "app", appName, "animal").AssertSuccess().AssertStdoutContains("giraffe")

	// now should exist, this should work too
	h.WithFile("app.yaml", fmt.Sprintf(`
name: %s
annotations:
  oracle.com/oci/subnetIds:
  - %s
config:
  animal: giraffe
  tea: oolong
`, appName, h.GetSubnetID()), 0644)
	h.Fn("deploy", "--all").AssertSuccess()
	h.Fn("invoke", appName, fnName).AssertSuccess()
	// make sure config was updated
	h.Fn("get", "config", "app", appName, "tea").AssertSuccess().AssertStdoutContains("oolong")

	// TODO we could test flag precedence of name here too
}

func TestDeployCreateApp(t *testing.T) {
	t.Parallel()

	h := testharness.Create(t)
	defer h.Cleanup()

	appName := h.NewAppName()
	fnName := h.NewFuncName(appName)
	h.MkDir(fnName)
	h.Cd(fnName)
	withMinimalFunction(h)
	withMinimalOCIApplication(h)
	h.Fn("deploy", "--app", appName, "--create-app").AssertSuccess()
}

func TestBump(t *testing.T) {
	t.Parallel()

	h := testharness.Create(t)
	defer h.Cleanup()

	expectFuncYamlVersion := func(v string) {
		funcYaml := h.GetFile("func.yaml")
		if !strings.Contains(funcYaml, fmt.Sprintf("version: %s", v)) {
			t.Fatalf("Exepected version to be %s but got %s", v, funcYaml)
		}

	}

	appName := h.NewAppName()
	withMinimalOCIApplication(h)
	h.Fn("create", "app", appName, "--annotation", h.GetSubnetAnnotation()).AssertSuccess()
	fnName := h.NewFuncName(appName)
	h.MkDir(fnName)
	h.Cd(fnName)
	withMinimalFunction(h)

	expectFuncYamlVersion("0.0.1")

	h.Fn("bump").AssertSuccess()
	expectFuncYamlVersion("0.0.2")

	h.Fn("bump", "--major").AssertSuccess()
	expectFuncYamlVersion("1.0.0")

	h.Fn("bump").AssertSuccess()
	expectFuncYamlVersion("1.0.1")

	h.Fn("bump", "--minor").AssertSuccess()
	expectFuncYamlVersion("1.1.0")

	h.Fn("deploy", "--app", appName).AssertSuccess()
	expectFuncYamlVersion("1.1.1")

	h.Fn("i", "function", appName, fnName).AssertSuccess().AssertStdoutContains(fmt.Sprintf(`%s:1.1.1`, fnName))

	h.Fn("deploy", "--no-bump", "--app", appName).AssertSuccess()
	expectFuncYamlVersion("1.1.1")

	h.Fn("i", "function", appName, fnName).AssertSuccess().AssertStdoutContains(fmt.Sprintf(`%s:1.1.1`, fnName))

}

// test fn list id value matches fn inspect id value for app, function & trigger
func TestListID(t *testing.T) {
	t.Parallel()

	h := testharness.Create(t)
	defer h.Cleanup()

	// execute create app, func & trigger commands
	appName := h.NewAppName()
	funcName := h.NewFuncName(appName)
	triggerName := h.NewTriggerName(appName, funcName)
	withMinimalFunction(h)
	withMinimalOCIApplication(h)
	h.Fn("create", "app", appName, "--annotation", h.GetSubnetAnnotation()).AssertSuccess()
	h.Fn("create", "function", appName, funcName, h.GetFnImage()).AssertSuccess()

	h.Fn("list", "apps", appName).AssertSuccess().AssertStdoutContains("ID")
	h.Fn("list", "fn", appName).AssertSuccess().AssertStdoutContains("ID")

	if !h.IsOCITestMode() {
		h.Fn("create", "trigger", appName, funcName, triggerName, "--type", "http", "--source", "/mytrigger").AssertSuccess()
		h.Fn("list", "triggers", appName).AssertSuccess().AssertStdoutContains("ID")
	}
}
