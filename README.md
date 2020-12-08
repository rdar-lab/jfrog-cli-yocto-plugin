# jfrog-cli-yocto-plugin

## About this plugin
This plugin allows integrating Yocto builds to Jfrog platform.

#### What is Yocto?
The Yocto Project (YP) is an open source collaboration project that helps developers create custom Linux-based systems regardless of the hardware architecture.

The project provides a flexible set of tools and a space where embedded developers worldwide can share technologies, software stacks, configurations, and best practices that can be used to create tailored Linux images for embedded and IOT devices, or anywhere a customized Linux OS is needed.  

#### Why to use Jfrog platform with Yocto?
With the jfrog platform you can control the lifecycle and distribution of the IoT firmware 

#### How to use this plugin?
 This plugin will invoke the bitbake process and will then upload the build results to artifactory. 
 It features:
 * Easy integration with one command to invoke the full process
 * Ability to deploy the artifacts to Artifactory and scan with Xray 

This plugin supports the following options:
* One command, clean, build, deploy **[ default, no flags are required ]**
* Continue build after failure without clean **[ use flag --clean=false ]**
* Build externally, use plugin only to deploy the artifacts **[ use flag --build=false ]**
* Build only, do not deploy to RT afterwards **[ use flag --load=false ]** 

## Using with docker

Using it with docker make it easier.
The docker image is based on gmacario/build-yocto which provides all the build-in tools for yocto build.

### instructions:
    1. build the project
        CMD:  docker build -t jfrog-yocto-builder .
    2. git clone your yocto project src files to your local machine. 
        Example: git clone -b dunfell git://git.yoctoproject.org/poky
    3. Make sure you have a proper artifactory configururation on your running machine.
        Use 'jfrog rt config' to configure your server IP and authentication
    4. RUN: docker run --rm -it jfrog-yocto-builder \
                   -v [YOCTO_WORKSPACE_LOCAL_FOLDER]:/home/build/workspace \
                   -v [USER_HOMEDIR]/.jfrog:/home/build/.frog \
                    core-image-minimal

## Installation with JFrog CLI
Since this plugin is currently not included in [JFrog CLI Plugins Registry](https://github.com/jfrog/jfrog-cli-plugins-reg), it needs to be built and installed manually. Follow these steps to install and use this plugin with JFrog CLI.
1. Make sure JFrog CLI is installed on you machine by running ```jfrog```. If it is not installed, [install](https://jfrog.com/getcli/) it.
2. Create a directory named ```plugins``` under ```~/.jfrog/``` if it does not exist already.
3. Clone this repository.
4. CD into the root directory of the cloned project.
5. Run ```go build``` to create the binary in the current directory.
6. Copy the binary into the ```~/.jfrog/plugins``` directory.

## Usage
### Commands
* configure
    - Used to configure the artifactory instance connection details. 
      This is an interactive command.
* bake
    - Used to build/deploy your yocto firmware 
    - Arguments:
        - target - The bake target. Examples: core-image-base, core-image-minimal
    - Flags:
        - run-folder - The location of the root folder to run the process from. **[Default: curr directory]**
        - build-env - The name of the "oe-init-build-env" to init the build env from. **[Default: oe-init-build-env]**
        - clean: Clean before the build, and clean the build-info on start **[Default: true]**
        - build: Perform a build. should be true unless you want to bypass and manually build **[Default: true]**
        - load: Load the resulting build to artifactory **[Default: true]**
        - scan: Scan the result with Xray **[Default: false]**
        - repo: Target repository to deploy to **[Default: yocto]**
        - artifact-name: Target name to deploy on
        - build-name: The name of the build
        - build-num: The number of the build
        - only-images: Deploy only the images of the build. **[Default: true]**
        - art-id: The server ID for artifactory configuration **[Default: using default config]**
            
    - Example:
    ```
  $ jfrog jfrog-cli-yocto-plugin bake core-image-minimal
  
    Running pre steps. Running directory=./
    Running Bit bake. target=core-image-minimal. This may take a long time....
    Loading the result to Artifactory

  ```

### Environment variables
None.

## Screenshots
##### Build artifacts
![Alt text](doc/img/artifacts.png?raw=true "Artifacts")
##### Build dependencies
![Alt text](doc/img/dependencies.png?raw=true "Artifacts")

## Additional info
Future improvements:
* Using artifactory as sstate cache server
* Xray scanning support
* Hierarchical dependency graph
