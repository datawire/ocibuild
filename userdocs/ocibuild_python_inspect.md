## ocibuild python inspect

Dump information about a Python environment

### Synopsis

Inspect a Python environment, and dump information about it for consumption by `ocibuild python wheel --platform-file=`.  The output also includes some informative fields that are not used by `ocibuild python wheel`.

LIMITATION: The --imagefile flag requires interacting with a running Docker.

```
ocibuild python inspect [flags] >PYTHON_PLATFORM.yml
```

### Options

```
  -h, --help                 help for inspect
      --imagefile string     Inspect a Docker image's Python rather than the host's Python
      --interpreter string   The Python interpreter to inspect (default "python3")
```

### SEE ALSO

* [ocibuild python](ocibuild_python.md)	 - Interact with Python without the target environment

