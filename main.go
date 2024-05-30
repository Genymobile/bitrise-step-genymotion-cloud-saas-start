package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-steputils/tools"
	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
)

// Define Genymotion constants
const (
	GMCloudSaaSInstanceUUID          = "GMCLOUD_SAAS_INSTANCE_UUID"
	GMCloudSaaSInstanceADBSerialPort = "GMCLOUD_SAAS_INSTANCE_ADB_SERIAL_PORT"
)

// Define variable
var isError bool = false

// Config ...
type Config struct {
	GMCloudSaaSEmail    string          `env:"email"`
	GMCloudSaaSPassword stepconf.Secret `env:"password"`
	GMCloudSaaSAPIToken stepconf.Secret `env:"api_token"`

	GMCloudSaaSRecipeUUID    string `env:"recipe_uuid,required"`
	GMCloudSaaSAdbSerialPort string `env:"adb_serial_port"`
	GMCloudSaaSGmsaasVersion string `env:"gmsaas_version"`
}

type Instance struct {
	UUID       string `json:"uuid"`
	ADB_SERIAL string `json:"adb_serial"`
	NAME       string `json:"name"`
}

type Output struct {
	Instance  Instance   `json:"instance"`
	Instances []Instance `json:"instances"`
}

// install gmsaas if not installed.
func ensureGMSAASisInstalled(version string) error {
	path, err := exec.LookPath("gmsaas")
	if err != nil {
		log.Infof("Installing gmsaas...")

		var installCmd *exec.Cmd
		if version != "" {
			installCmd = exec.Command("pipx", "install", "gmsaas=="+version)
		} else {
			installCmd = exec.Command("pipx", "install", "gmsaas")
		}

		if out, err := installCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%s failed, error: %s | output: %s", installCmd.Args, err, out)
		}

		// Execute asdf reshim pour update PATH
		exec.Command("asdf", "reshim", "python").CombinedOutput()

		// Execute pipx ensurepath to update PATH
		exec.Command("pipx", "ensurepath").CombinedOutput()

		if version != "" {
			log.Infof("gmsaas %s has been installed.", version)
		} else {
			log.Infof("gmsaas has been installed.")
		}

	} else {
		log.Infof("gmsaas is already installed: %s", path)
	}

	// Set Custom user agent to improve customer support
	os.Setenv("GMSAAS_USER_AGENT_EXTRA_DATA", "bitrise.io")
	return nil
}

// printError prints an error.
func printError(format string, args ...interface{}) {
	log.Errorf(format, args...)
}

// abortf prints an error and terminates step
func abortf(format string, args ...interface{}) {
	printError(format, args...)
	os.Exit(1)
}

// setOperationFailed marked step as failed
func setOperationFailed(format string, args ...interface{}) {
	printError(format, args...)
	isError = true
}

func getADBSerialFromJSON(jsonData string) string {
	var output Output
	if err := json.Unmarshal([]byte(jsonData), &output); err != nil {
		setOperationFailed("Issue with JSON parsing : %w", err)
	}
	return output.Instance.ADB_SERIAL
}

func getInstanceDetails(name string) (string, string) {
	cmd := command.New("gmsaas", "--format", "json", "instances", "list")
	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		setOperationFailed("Failed to get instances list, error: error: %s | output: %s", cmd.PrintableCommandArgs(), err, out)
		return "", ""
	}
	var output Output
	if err := json.Unmarshal([]byte(out), &output); err != nil {
		setOperationFailed("Issue with JSON parsing : %w", err)
	}

	for _, instance := range output.Instances {
		if instance.NAME == name {
			return instance.UUID, instance.ADB_SERIAL
		}
	}
	return "", ""
}

func configureAndroidSDKPath() {
	log.Infof("Configure Android SDK configuration")

	value, exists := os.LookupEnv("ANDROID_HOME")
	if exists {
		cmd := command.New("gmsaas", "config", "set", "android-sdk-path", value)
		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		if err != nil {
			setOperationFailed("Failed to set android-sdk-path, error: error: %s | output: %s", cmd.PrintableCommandArgs(), err, out)
			return
		}
		log.Infof("Android SDK is configured")
	} else {
		setOperationFailed("Please set ANDROID_HOME environment variable")
		return
	}
}

func login(api_token, username, password string) {
	log.Infof("Login Genymotion Account")

	var cmd *exec.Cmd
	if api_token != "" {
		cmd = exec.Command("gmsaas", "auth", "token", api_token)
	} else if username != "" && password != "" {
		cmd = exec.Command("gmsaas", "auth", "login", username, password)
	} else {
		abortf("Invalid arguments. Must provide either a token or both email and password.")
		return
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		abortf("Failed to log with gmsaas, error: error: %s | output: %s", cmd.Args, err, out)
		return
	}

	log.Infof("Logged to Genymotion Cloud SaaS platform")
}

func startInstanceAndConnect(wg *sync.WaitGroup, recipeUUID, instanceName, adbSerialPort string) {
	var output Output
	defer wg.Done()
	cmd := command.New("gmsaas", "--format", "json", "instances", "start", recipeUUID, instanceName)
	jsonData, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		setOperationFailed("Failed to start a device, error: %s | output: %s\n", err, jsonData)
		return
	}

	if err := json.Unmarshal([]byte(jsonData), &output); err != nil {
		setOperationFailed("Issue with JSON parsing : %s", err)
	}

	// Connect to adb with adb-serial-port
	if adbSerialPort != "" {
		cmd := command.New("gmsaas", "--format", "json", "instances", "adbconnect", output.Instance.UUID, "--adb-serial-port", adbSerialPort)
		ADBjsonData, err := cmd.RunAndReturnTrimmedCombinedOutput()
		if err != nil {
			setOperationFailed("Failed to connect a device, error: error: %s | output: %s", cmd.PrintableCommandArgs(), err, ADBjsonData)
			return
		}
		if err := json.Unmarshal([]byte(ADBjsonData), &output); err != nil {
			setOperationFailed("Issue with JSON parsing : %s", err)
		}
	} else {
		cmd := command.New("gmsaas", "--format", "json", "instances", "adbconnect", output.Instance.UUID)
		ADBjsonData, err := cmd.RunAndReturnTrimmedCombinedOutput()
		if err != nil {
			setOperationFailed("Failed to connect a device, error: error: %s | output: %s", cmd.PrintableCommandArgs(), err, ADBjsonData)
			return
		}
		if err := json.Unmarshal([]byte(ADBjsonData), &output); err != nil {
			setOperationFailed("Issue with JSON parsing : %s", err)
		}
	}

	log.Infof("Genymotion instance UUID : %s has been started and connected with ADB Serial Port : %s", output.Instance.UUID, output.Instance.ADB_SERIAL)

}

func main() {

	var c Config
	if err := stepconf.Parse(&c); err != nil {
		abortf("Issue with input: %s", err)
	}
	stepconf.Print(c)

	if err := ensureGMSAASisInstalled(c.GMCloudSaaSGmsaasVersion); err != nil {
		abortf("%s", err)
	}
	configureAndroidSDKPath()

	if err := tools.ExportEnvironmentWithEnvman("GMSAAS_USER_AGENT_EXTRA_DATA", "bitrise.io"); err != nil {
		printError("Failed to export %s, error: %v", "GMSAAS_USER_AGENT_EXTRA_DATA", err)
	}

	if c.GMCloudSaaSAPIToken != "" {
		login(string(c.GMCloudSaaSAPIToken), "", "")
	} else {
		login("", c.GMCloudSaaSEmail, string(c.GMCloudSaaSPassword))
	}

	instancesList := []string{}
	adbSerialList := []string{}
	adbSerialPortList := []string{}

	recipesList := strings.Split(c.GMCloudSaaSRecipeUUID, ",")

	if len(c.GMCloudSaaSAdbSerialPort) >= 1 {
		adbSerialPortList = strings.Split(c.GMCloudSaaSAdbSerialPort, ",")
	}

	buildNumber := os.Getenv("BITRISE_BUILD_NUMBER")

	log.Infof("Start %d Android instances on Genymotion Cloud SaaS", len(recipesList))
	var wg sync.WaitGroup
	for cptInstance := 0; cptInstance < len(recipesList); cptInstance++ {
		instanceName := fmt.Sprint("gminstance_bitrise_", buildNumber, "_", cptInstance)
		wg.Add(1)
		if len(adbSerialPortList) >= 1 {
			go startInstanceAndConnect(&wg, recipesList[cptInstance], instanceName, adbSerialPortList[cptInstance])
		} else {
			go startInstanceAndConnect(&wg, recipesList[cptInstance], instanceName, "")
		}
	}
	wg.Wait()

	for cptInstance := 0; cptInstance < len(recipesList); cptInstance++ {

		instanceName := fmt.Sprint("gminstance_bitrise_", buildNumber, "_", cptInstance)
		instanceUUID, InstanceADBSerialPort := getInstanceDetails(instanceName)

		instancesList = append(instancesList, instanceUUID)
		adbSerialList = append(adbSerialList, InstanceADBSerialPort)
	}

	// --- Step Outputs: Export Environment Variables for other Steps:
	outputs := map[string]string{
		GMCloudSaaSInstanceUUID:          strings.Join(instancesList, ","),
		GMCloudSaaSInstanceADBSerialPort: strings.Join(adbSerialList, ","),
	}

	for k, v := range outputs {
		if err := tools.ExportEnvironmentWithEnvman(k, v); err != nil {
			abortf("Failed to export %s, error: %v", k, err)
		}
	}

	// --- Exit codes:
	// The exit code of your Step is very important. If you return
	//  with a 0 exit code `bitrise` will register your Step as "successful".
	// Any non zero exit code will be registered as "failed" by `bitrise`.
	if isError {
		// If at least one error happens, step will fail
		os.Exit(1)
	}
	os.Exit(0)
}
