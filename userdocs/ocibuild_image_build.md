## ocibuild image build

Combine layers in to a complete image

```
ocibuild image build [flags] IN_LAYERFILES... >OUT_IMAGEFILE
```

### Options

```
      --base IN_IMAGEFILE                     Use IN_IMAGEFILE as the base of the image
  -c, --config.Cmd command                    Set the resulting image's command
      --config.Entrypoint entrypoint          Set the resulting image's entrypoint
  -e, --config.Env.append KEY=VALUE           Append KEY=VALUE in the resulting image's environment
  -E, --config.Env.clear                      Discard any environment variables set in the base image's config
  -w, --config.WorkingDir working-directory   Set the resulting image's working-directory
  -h, --help                                  help for build
  -t, --tag TAG                               Tag the resulting image as TAG
```

### SEE ALSO

* [ocibuild image](ocibuild_image.md)	 - Manipulate complete images

