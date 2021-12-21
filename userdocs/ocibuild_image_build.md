## ocibuild image build

Combine layers in to a complete image

```
ocibuild image build [flags] IN_LAYERFILES... >OUT_IMAGEFILE
```

### Options

```
      --base IN_IMAGEFILE       Use IN_IMAGEFILE as the base of the image
  -e, --entrypoint Entrypoint   Set the resulting image's Entrypoint
  -h, --help                    help for build
  -t, --tag TAG                 Tag the resulting image as TAG
```

### SEE ALSO

* [ocibuild image](ocibuild_image.md)	 - Manipulate complete images

