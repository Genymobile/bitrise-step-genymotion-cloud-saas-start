package main

import (
	"bufio"
	"os"
	"os/exec"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-tools/go-steputils/stepconf"
)

const (
	genymotionCloudSaaSInstanceUUID          = "GENYMOTION_CLOUD_SAAS_INSTANCE_UUID"
	genymotionCloudSaaSInstanceADBSerialPort = "GENYMOTION_CLOUD_SAAS_INSTANCE_ADB_SERIAL_PORT"
)

// Config ...
type Config struct {
	GenymotionCloudLogin    string `env:"genymotion_cloud_saas_login,required"`
	GenymotionCloudPassword string `env:"genymotion_cloud_saas_password,required"`

	GenymotionCloudRecipeUUID    string `env:"genymotion_cloud_saas_recipe_uuid,required"`
	GenymotionCloudInstanceName  string `env:"genymotion_cloud_saas_instance_name,required"`
	GenymotionCloudAdbSerialPort string `env:"genymotion_cloud_saas_adb_serial_port"`
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
	out, _ := adminList.StdoutPipe()
	err := adminList.Start()
	if err != nil {
		failf("Issue with gmssas command line: %s", err)
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

func checkAndroidSDKPath() {
	log.Infof("Check Android SDK configuration")
	cmd := exec.Command("gmsaas", "config", "get", "android-sdk-path")
	stdout, _ := cmd.CombinedOutput()
	if strings.Compare(strings.TrimRight(string(stdout), "\n"), "None") == 0 {
		failf("Please configure android-sdk-path, you can find more information here: https://docs.genymotion.com/saas/latest/08_gmsaas.html#configuration")
	} else {
		log.Infof("Android SDK is configured")
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

func exportEnvironmentWithEnvman(keyStr, valueStr string) error {
	cmd := command.New("envman", "add", "--key", keyStr)
	cmd.SetStdin(strings.NewReader(valueStr))
	return cmd.Run()
}

func main() {

	var c Config
	if err := stepconf.Parse(&c); err != nil {
		failf("Issue with input: %s", err)
	}
	stepconf.Print(c)

	checkAndroidSDKPath()

	login(c.GenymotionCloudLogin, c.GenymotionCloudPassword)

	log.Infof("Start Android devices on Genymotion Cloud SaaS")
	cmd := exec.Command("gmsaas", "instances", "start", c.GenymotionCloudRecipeUUID, c.GenymotionCloudInstanceName)
	stdout, err := cmd.CombinedOutput()
	if err != nil {
		failf("Failed to start a device, error: %#v | output: %s", err, stdout)
	} else {
		log.Infof("Device started %s", stdout)
	}

	instanceUUID := strings.TrimRight(string(stdout), "\n")

	// Connect to adb with adb-serial-port
	if c.GenymotionCloudAdbSerialPort != "" {
		cmd := exec.Command("gmsaas", "instances", "adbconnect", instanceUUID, "--adb-serial-port", c.GenymotionCloudAdbSerialPort)
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

	_, InstanceADBSerialPort := getInstanceDetails(c.GenymotionCloudInstanceName)

	// --- Step Outputs: Export Environment Variables for other Steps:
	outputs := map[string]string{
		genymotionCloudSaaSInstanceUUID:          instanceUUID,
		genymotionCloudSaaSInstanceADBSerialPort: InstanceADBSerialPort,
	}

	for k, v := range outputs {
		if err := exportEnvironmentWithEnvman(k, v); err != nil {
			failf("Failed to export %s, error: %v", k, err)
		}
	}

	// --- Exit codes:
	// The exit code of your Step is very important. If you return
	//  with a 0 exit code `bitrise` will register your Step as "successful".
	// Any non zero exit code will be registered as "failed" by `bitrise`.
	os.Exit(0)
}
