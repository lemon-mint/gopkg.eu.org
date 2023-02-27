# gopkg.eu.org Module Index

Go modules hosted on gopkg.eu.org

## Add a module

To add a module to the index, fork this repository, pick a unique name for your module and add a file named `modname` to the `modules` directory. The file should contain the following content:

```yaml
# full module name
root: gopkg.eu.org/broccoli
# the VCS used to fetch the module (default: git)
vcs: git
# the URL to fetch the module from
url: https://github.com/unsafe-risk/broccoli.git
# the description of the module
description: Simple CLI Package for Go
```

then create a pull request.
