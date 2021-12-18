## ocibuild layer wheel

Turn a Python wheel in to a layer

### Synopsis

Given a Python wheel file, transform it in to a layer.

In order to transform the wheel in to a layer, ocibuild needs to know a few things about the target environment.  You must supply this to ocibuild using the --platform-file flag, pointing it at a YAML file that is as follows:

    # file locations
    ConsoleShebang: /usr/bin/python3.9
    GraphicalShebang: /usr/bin/python3.9
    # You can obtain the scheme paths for a running Python instance with
    #     import json
    #     from pip._internal.locations import get_scheme
    #     scheme=get_scheme(')
    #     print(json.dumps({slot: getattr(scheme, slot) for slot in scheme.__slots__}))
    Scheme:
      purelib: /usr/lib/python3.9/site-packages
      platlib: /usr/lib/python3.9/site-packages
      headers: /usr/include/site/python3.9/
      scripts: /usr/bin
      data: /usr

    # user account
    UID: 0
    GID: 0
    UName: root
    GName: root

    # command to run on the host (not target) system to generate .pyc
    # files.  The Python version number must match the target Python's
    # version number rather precisely; or rather their
    # `importlib.util.MAGIC_NUMBER` values must match.
    PyCompile: ['python3.9', '-m', 'compileall']

LIMITATION: It is 'TODO' to create an 'ocibuild python WHATEVER' command that can inspect an image's Python installation and emit the appropriate YAML description of it.

```
ocibuild layer wheel [flags] IN_WHEELFILE.whl >OUT_LAYERFILE
```

### Options

```
  -h, --help                         help for wheel
      --platform-file IN_YAML_FILE   Read IN_YAML_FILE to determine details about the target platform
```

### SEE ALSO

* [ocibuild layer](ocibuild_layer.md)	 - Manipulate individual layers for use in an image

