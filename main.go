package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

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

// Config ...
type Config struct {
	GMCloudSaaSEmail    string          `env:"email,required"`
	GMCloudSaaSPassword stepconf.Secret `env:"password,required"`

	GMCloudSaaSRecipeUUID    string `env:"recipe_uuid,required"`
	GMCloudSaaSInstanceName  string `env:"instance_name,required"`
	GMCloudSaaSAdbSerialPort string `env:"adb_serial_port"`
}

// install gmsaas if not installed.
func ensureGMSAASisInstalled() error {
	path, err := exec.LookPath("gmsaas")
	if err != nil {
		log.Infof("Installing gmsaas ...")
		cmd := command.New("pip3", "install", "gmsaas")
		if out, err := cmd.RunAndReturnTrimmedCombinedOutput(); err != nil {
			return fmt.Errorf("%s failed, error: %s | output: %s", cmd.PrintableCommandArgs(), err, out)
		}
		log.Infof("gmsaas has been installed.")
	} else {
		log.Infof("gmsaas is already installed : %s", path)
	}
	return nil
}

// failf prints an error and terminates the step.
func failf(format string, args ...interface{}) {
	log.Errorf(format, args...)
	os.Exit(1)
}

func getInstanceDetails(name string) (string, string) {
	for index, line := range getInstancesList() {
		if index >= 2 {
			s := strings.Split(line, "  ")
			if strings.Compare(s[1], name) == 0 {
				uuid := s[0]
				serial := s[2]
				return uuid, serial
			}
		}
	}
	return "", ""
}

func getInstancesList() []string {
	adminList := exec.Command("gmsaas", "instances", "list")
	out, err := adminList.StdoutPipe()
	if err != nil {
		failf("Issue with gmsaas command line: %s", err)
	}
	if err := adminList.Start(); err != nil {
		failf("Issue with gmsaas command line: %s", err)
	}
	// Create new Scanner.
	scanner := bufio.NewScanner(out)
	result := []string{}
	// Use Scan.
	for scanner.Scan() {
		line := scanner.Text()
		// Append line to result.
		result = append(result, line)
	}
	return result
}

func configureAndroidSDKPath() {
	log.Infof("Configure Android SDK configuration")

	value, exists := os.LookupEnv("ANDROID_HOME")
	if exists {
		cmd := exec.Command("gmsaas", "config", "set", "android-sdk-path", value)
		stdout, err := cmd.CombinedOutput()
		if err != nil {
			failf("Fail to set android-sdk-path, error: %#v | output: %s", err, stdout)
		}
		log.Infof("Android SDK is configured")
	} else {
		failf("Please set ANDROID_HOME environment variable")
	}
}

func login(username, password string) {
	log.Infof("Login Genymotion Account")
	cmd, err := exec.Command("gmsaas", "auth", "login", username, password).CombinedOutput()
	if err != nil {
		failf("Failed to log with gmsaas, error: %#v | output: %s", err, cmd)
	} else {
		log.Infof("Logged to Genymotion Cloud SaaS platform")
	}
}

func main() {

	var c Config
	if err := stepconf.Parse(&c); err != nil {
		failf("Issue with input: %s", err)
	}
	stepconf.Print(c)

	if err := ensureGMSAASisInstalled(); err != nil {
		failf("%s", err)
	}
	configureAndroidSDKPath()

	login(c.GMCloudSaaSEmail, string(c.GMCloudSaaSPassword))

	log.Infof("Start Android devices on Genymotion Cloud SaaS")
	cmd := exec.Command("gmsaas", "instances", "start", c.GMCloudSaaSRecipeUUID, c.GMCloudSaaSInstanceName)
	stdout, err := cmd.CombinedOutput()
	if err != nil {
		failf("Failed to start a device, error: %#v | output: %s", err, stdout)
	} else {
		log.Infof("Device started %s", stdout)
	}

	instanceUUID := strings.TrimRight(string(stdout), "\n")

	// Connect to adb with adb-serial-port
	if c.GMCloudSaaSAdbSerialPort != "" {
		cmd := exec.Command("gmsaas", "instances", "adbconnect", instanceUUID, "--adb-serial-port", c.GMCloudSaaSAdbSerialPort)
		output, err := cmd.CombinedOutput()
		if err != nil {
			cmd = exec.Command("gmsaas", "instances", "stop", instanceUUID)
			log.Errorf("Device stopped %s", instanceUUID)
			failf("Error: %s", output)
		}
	} else {
		cmd := exec.Command("gmsaas", "instances", "adbconnect", instanceUUID)
		output, err := cmd.CombinedOutput()
		if err != nil {
			cmd = exec.Command("gmsaas", "instances", "stop", instanceUUID)
			log.Errorf("Device stopped %s", instanceUUID)
			failf("Error: %s", output)
		}
	}

	_, InstanceADBSerialPort := getInstanceDetails(c.GMCloudSaaSInstanceName)

	// --- Step Outputs: Export Environment Variables for other Steps:
	outputs := map[string]string{
		GMCloudSaaSInstanceUUID:          instanceUUID,
		GMCloudSaaSInstanceADBSerialPort: InstanceADBSerialPort,
	}

	for k, v := range outputs {
		if err := tools.ExportEnvironmentWithEnvman(k, v); err != nil {
			failf("Failed to export %s, error: %v", k, err)
		}
	}

	// --- Exit codes:
	// The exit code of your Step is very important. If you return
	//  with a 0 exit code `bitrise` will register your Step as "successful".
	// Any non zero exit code will be registered as "failed" by `bitrise`.
	os.Exit(0)
}
