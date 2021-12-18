## ocibuild python getwheel

Download a wheel file from the Python Package Index

### Synopsis

Given a wheel filename, download it from a package index, writing the file contents to stdout.

LIMITATION: Generating the list of wheel files to download is non-obvious at this point; soon there will be an `ocibuild python SOMETHING` command that will spit out a list of wheel filenames, but it doesn't exist yet.  I'm not sure if you can get pip to give it to you.  pip-compile only gives you (name, version) tuples, not the full (name, version, platform) tuple.

LIMITATION: While checksums are verified, GPG signatures are not.

```
ocibuild python getwheel [flags] NAME_VERSION_PLATFORM.whl >NAME_VERSION_PLATFORM.whl
```

### Options

```
  -h, --help                  help for getwheel
      --index-server string   Index server to download the wheel from (default "https://pypi.org/simple/")
```

### SEE ALSO

* [ocibuild python](ocibuild_python.md)	 - Interact with Python without the target environment

