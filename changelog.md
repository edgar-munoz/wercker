## unreleased

- docker-build bugfix: if the dockerfile build fails then the error is displayed and the step fails (#424)

## v1.0.1264 (2015-06-18)

- Collect report artifacts from a step even if it failed (#428)

## v1.0.1260 (2018-06-14)
- Changes to access RDD API Service and inject RDD in build pipelines (#421)
- Fix wercker --help (#426)

## v1.0.1244 (2018-05-29)
- Changes for WRKR-347 Allow switching between app/dev sites (#419)

## v1.0.1238 (2018-05-23)
- Wercker runner config (#417)
- Docker file integration (#415)

## v1.0.1230 (2018-05-15)

- Support for publishing private steps (#409)
- Wercker CLI changes for external runner (#406)
    Changes for WRKR-76 and WRKR-207
    Adding code to format json log entries, write log to disk
    Clean up logging to disk
    Cleanup logging when JobId encountered.
    Fix --nowait option not working
    Fix to use proper logger
    Allow store path as env variable WERCKER_RUNNER_STOREPATH. Make sure directory is created if doesn't exist
    Add informational message telling user where local output is stored.
    Runner configure without remote repository pull
    * Changes for WRKR-76, WRKR-77, and WRKR-207
    * Fix typo in flags definition

## v1.0.1226 (2018-05-11)

- Reverted Cosmetic changes for docker file integration (#407)
- Reverted Docker file integration, docker-networks, docker-run, docker-kill (#405)

## v1.0.1223 (2018-05-08)

- External runner changes removed by prior revert (merge pull request #384) 
- Cosmetic changes for docker file integration (#407)
- Docker file integration, docker-networks, docker-run, docker-kill (#405)

## v1.0.1216 (2018-05-01)

- Add Oracle Contributor Agreement (#400)
- Convert docker-push to use the official docker client (#399)

## v1.0.1210 (2018-04-25)

- Revert docker links to docker networks change, which was causing build issues (#397)

## v1.0.1205 (2018-04-20)

- Changes for robust error handling and reporting in docker-push (#387)
- Replace docker links with docker network (#382)
- Change some docker API calls to use the official Docker client (#385) 

## v1.0.1201 (2018-04-16)

- Update azure client to allow docker-push in all regions (#381)

## v1.0.1196(2018-04-11)

- Fixes and additional properties for internal/docker-build step (#372)

## v1.0.1195(2018-04-10)

- Fix for correctly inferring regsitry and repoistory from step inputs (#375) 
- Fix "go build" and "wercker build" on golang 1.10 (#374)

## v1.0.1189(2018-04-04)

- Fix status reporting for docker push (#371)

## v1.0.1183 (2018-03-29)

- New docker-build step and enhanded docker-push step (#362)

## v1.0.1153 (2018-02-27)

- Remove Keen dependencies (#354)

## v1.0.1062 (2017-11-28)

- Default docker hub push to registry V2 (#348)

## v1.0.1049 (2017-11-15)

- Update dependencies, as a result of `Sirupsen/logrus` -> `sirupsen/logrus` (#333)
- Add a Docker subcommand (#335)
- Ensure repository names are always lowercase (#338)
- Support for the new step manifest format (#343)

## v1.0.965 (2017-08-23)

- Change compilation in separate wercker steps (#331)
- Add retry and exponential backoff for fetching step metadata and step tarball
  (#330)
- Add flag to delete Docker image after pushing it to a registry (#327)
- Use wercker registry for wercker-init (#334)

## v1.0.938 (2017-07-27)

- Some nice additions to the way we do the docker push and test (#320)
- Fix env var loading order (#314, #315, #317)
- Fix internal/watchstep (#312)
- Add env option to docker-scratch-push (#295)
- Allow relative paths for file:// targets in dev mode (#296)
- Better control limiting memory on run containers, when using
  services gives the services a 25% of the total memory to split
  amongst themselves, defaults to no limits (#299)
- Automatically detect bash or sh for containers by default,
  defaulting to bash if it is there (#301)
- Fix a small bug when doing local deploys and using a working-dir other
  than .wercker (#301)

## v1.0.758 (2017-01-27)

- Add Azure Registry support (#275)
- Explicitly chmods the basepath / source path to a+rx
- Removes the explicit clear after launching a shell (#257)
- Fix `wercker doc` and update `./Documentation/*` (#260)

## v1.0.643 (2016-10-05)

- Remove google as default container DNS (#245)
- Update to compiling with go 1.7

## v1.0.629 (2016-09-21)

- Add additional output when storing artifacts (#207)
- Fix longer (2+) chains of runs that have source-dir specified (#151)
- Output more descriptive error message when setup environment fails (#230)
- Allow use of an "ignore-file" yaml directive that parse the gitignore syntax
  (#240)

## v1.0.560 (2016-07-14)

- Fix internal/docker-scratch-push for Docker 1.10+

## v1.0.547 (2016-07-01)

- Add checkpointing and base-path (#123)
- Support for registry v2 (#131)
- Mount volumes in the container from different local paths (#134)
- Only push tags that were defined in the wercker.yml (#142)
- wercker is now using govendor (#146)
- Display raw config, before parsing it (#149)
- Allow multiple services with the same images (#159)
- Add exposed-ports (#161)
- Fix run, build and deploy urls (#163)

## 2016.03.11

### Features

- Moves the working path to default to `.wercker` and removes the flags
  for configuring the other paths
- Adds a symlink `.wercker/latest` for referring to your latest build, and
  a `.wercker/latest_deploy` for referring to your latest deploy
- Make the --artifacts work better locally, making your build's artifacts
  easily available under .wercker/latest/output
- Automatically use the contents of `.wercker/latest/output` when running a
  `wercker deploy` without specifying a target
- When running `wercker deploy` if the specified target does not container a
  wercker.yml file, attempt to use the one in the current directory.
- Allow settings multiple tags at a time when doing `internal/docker-push`
- Check for and allow unix:///var/run/docker.sock on non-linux hosts


### Bug Fixes

- Deal with symlinks significantly better
- Respect --docker-local when using `internal/docker-push` (don't push)
- Allow images to be pulled by nested local services (removes
  implicit --docker-local)
- Workaround a docker issue related to not fully consuming the result of a
  CopyFromContainer API call (when we exported a cache that was more than our
  limit of 1GB we'd just drop it, and docker would hang)
- Remove pipeline ID tag set by `internal/docker-push`


## 2016.02.10

### Features

- Allow users to mount local volumes to their wercker build containers, specified by a list of `volumes` underneath box in the werker.yml file. Must have `--enable-volumes` flag set in order to run.
- Check to see if config from wercker.yml is empty
- Adds changelog

### Bug fixes

- Fixes to the shellstep implementation
