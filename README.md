# jfrog-cli-yocto-plugin

## About this plugin
This plugin allows integrating Yocto builds to Jfrog platform

## Installation with JFrog CLI
Installing the latest version:

`$ jfrog plugin install jfrog-cli-yocto-plugin`

Installing a specific version:

`$ jfrog plugin install jfrog-cli-yocto-plugin@version`

Uninstalling a plugin

`$ jfrog plugin uninstall jfrog-cli-yocto-plugin`

## Usage
### Commands
* bake
    - Arguments:
        - run-folder - The location of the root folder to run the process from.
        - build-env - The location of the "oe-init-build-env" to init the build env from
        - target - The bake target. Examples: core-image-base, core-image-minimal
    - Flags:
        - clean: Clean before the build, and clean the build-info on start **[Default: true]**
        - build: Perform a build. should be true unless you want to bypass and manually build **[Default: true]**
        - load: Load the resulting build to artifactory **[Default: true]**
        - scan: Scan the result with Xray **[Default: false]**
        - repo: Target repository to deploy to **[Default: yocto]**
        - artifactName: Target name to deploy on
        - buildName: The name of the build
        - buildNum: The number of the build
        - onlyImages: Deploy only the images of the build. **[Default: true]**
            
    - Example:
    ```
  $ jfrog jfrog-cli-yocto-plugin bake ./ oe-init-build-env core-image-minimal
  
    Running pre steps. Running directory=./
    Running Bit bake. target=core-image-minimal. This may take a long time....
    Loading the result to Artifactory

  ```

### Environment variables
None.

## Additional info
None.
