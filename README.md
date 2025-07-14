

## About The Plugin
The cosmo plugin is writen to be use with kubectl to manage the installation of Cosmonic Control platform within Kubernetes that manages your webassembly components and providers.

## Getting Started
Copy the binary `kubectl-cosmo` to somewhere in your path. This allows for `kubectl` to find and use the plugin.

## Usage
Usage:
```
kubectl cosmo

Interact with Cosmonic Control

Usage:
  cosmo [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  console     launch the Cosmonic console
  docs        Open the default browser to https://cosmonic.com/docs
  help        Help about any command
  hostgroup   Manage hostgroups within the cluster
  license     To obtain a license, visit cosmonic.com and sign up for a free trial key
  nexus       Manage the Nexus Cosmonic control-plane
  version     Returns the versions of all resources installed for Cosmonic Control

Flags:
  -h, --help   help for cosmo
```

### Additional Usage Examples

- Launch the Cosmonic console:
  ```sh
  kubectl cosmo console
  ```

- Open Cosmonic documentation in your browser:
  ```sh
  kubectl cosmo docs
  ```

- Show installed resource versions:
  ```sh
  kubectl cosmo version
  ```

- Manage hostgroups:
  ```sh
  kubectl cosmo hostgroup [subcommand]
  ```

- Manage the Nexus control-plane:
  ```sh
  kubectl cosmo nexus [subcommand]
  ```

## Acknowledgements
The Krew kubectl plugin project
Used [sample-cli-plugin project](https://github.com/kubernetes/sample-cli-plugin/tree/master)
