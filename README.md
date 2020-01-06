# genymotion-cloud-saas-start

Start Genymotion Cloud SaaS Android devices

## Prerequisite

Go to  [Genymotion Cloud SaaS](https://cloud.geny.io/?&utm_source=web-referral&utm_medium=github&utm_campaign=bitrise&utm_content=signup) and create an account

## How to use this Step

Can be run directly with the [bitrise CLI](https://github.com/bitrise-io/bitrise),
just `git clone` this repository, `cd` into it's folder in your Terminal/Command Line
and call `bitrise run test`.

*Check the `bitrise.yml` file for required inputs which have to be
added to your `.bitrise.secrets.yml` file!*

Step by step:

1. Open up your Terminal / Command Line
2. `git clone` the repository
3. `cd` into the directory of the step (the one you just `git clone`d)
5. Create a `.bitrise.secrets.yml` file in the same directory of `bitrise.yml`
   (the `.bitrise.secrets.yml` is a git ignored file, you can store your secrets in it)
6. Check the `bitrise.yml` file for any secret you should set in `.bitrise.secrets.yml`
  * Best practice is to mark these options with something like `# define these in your .bitrise.secrets.yml`, in the `app:envs` section.
7. Once you have all the required secret parameters in your `.bitrise.secrets.yml` you can just run this step with the [bitrise CLI](https://github.com/bitrise-io/bitrise): `bitrise run test`

An example `.bitrise.secrets.yml` file:

```
envs:
- GMCLOUD_SAAS_EMAIL: [YOUR_GENYMOTION_CLOUD_EMAIL]
- GMCLOUD_SAAS_PASSWORD: [YOUR_GENYMOTION_CLOUD_PASSWORD]
```

## How to setup Bitrise.yml

This step takes three inputs:
  * `recipe_uuid`: Recipe UUID is the identifier used when starting an instance; it can be retrieved using `gmsaas recipes list`
  * `instance_name`: Name given to the newly created instance.
  * `adb_serial_port` (default value: None): port which the instance will be connected to ADB

Example: 

```
  inputs:
    - email: $GMCLOUD_SAAS_EMAIL
    - password: $GMCLOUD_SAAS_PASSWORD
    - instance_name: DeviceStartedByBitrise
    - recipe_uuid: e20da1a3-313c-434a-9d43-7268b12fee08
    - adb_serial_port: 4321
```
## See also

This step is part of a series of Bitrise steps which integrate Genymotion Cloud SaaS with Bitrise.

 * Use the [Stop Genymotion Cloud SaaS android devices](https://github.com/genymobile/bitrise-step-genymotion-cloud-saas-stop.git) step to stop your Android devices to Genymotion Cloud SaaS.

## How to contribute to this Step

1. Fork this repository
2. `git clone` it
3. Create a branch you'll work on
4. To use/test the step just follow the **How to use this Step** section
5. Do the changes you want to
6. Run/test the step before sending your contribution
  * You can also test the step in your `bitrise` project, either on your Mac or on [bitrise.io](https://www.bitrise.io)
  * You just have to replace the step ID in your project's `bitrise.yml` with either a relative path, or with a git URL format
  * (relative) path format: instead of `- original-step-id:` use `- path::./relative/path/of/script/on/your/Mac:`
  * direct git URL format: instead of `- original-step-id:` use `- git::https://github.com/user/step.git@branch:`
  * You can find more example of alternative step referencing at: https://github.com/bitrise-io/bitrise/blob/master/_examples/tutorials/steps-and-workflows/bitrise.yml
7. Once you're done just commit your changes & create a Pull Request

