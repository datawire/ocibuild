# Overview of file formats and the data models within

 > "The nice thing about standards is that you have so many to choose
 > from."

`ocibuild` interacts with images and layers as files.  This document
describes the format of those files (and the format of similar files
from other tools), and the data model being represented in those
files.

## Data model: v1 images and v2 images

TL;DR: They aren't well-defined terms, but there are two image data
models; we only use v2, and unless otherwise specified this document
only discusses v2.

In the olden days, Docker images were a stack of (layerdata, metadata)
tuples; an image ID was just the layer ID of the top layer.  Then
someone decided that actually there shouldn't be per-layer metadata,
and the image ID would be the metadata file (the "config" file) for
the image, and that metadata file would contain a list of layerdata
IDs.  Some folks colloquially call these the "v1" and "v2" image
formats.


         "v1" image                   "v2" image
         ==========                   ==========

       metadata:aaaa ---> layerdata:aaaa <--,
             ^                              |
       metadata:bbbb ---> layerdata:bbbb <--,
             ^                              |
       metadata:cccc ---> layerdata:cccc <--,
             ^                              |
       metadata:dddd ---> layerdata:dddd <--,
             ^                              |
       metadata:eeee ---> layerdata:eeee <--,
             ^                              |
             |                           config
             |                              ^
       image_id="eeee"           image_id=hash(config)

While these are colloquially "v1" and "v2" image data models, they
actually correspond to the [Docker Image][] Specification
[v1.0.0][Docker Image v1.0.0] and [v1.1.0+][Docker Image v1.1.0]
respectively.  Actually, v1.1.0 specifies a file format that contains
both the old v1 data model and the new v2 data model; so that files
emitted by new versions of Docker (with `docker save`) could still be
consumed by old versions of Docker (with `docker load`).

This v2 data model is what got adopted by the Open Container
Initiative (OCI) and is what this document will be discussing.  The
remainder of this document will be ignoring the v1 data model unless
otherwise specified.

## File format concepts

A key idea in the modern Docker/OCI ecosystem is "content-addressable
storage"; that everything forms a Merkle DAG.  Think "git: referring
to content by its hash".

In Docker/OCI, each piece of content is called a "blob".  The content
of a blob file is the raw content being stored.  The ID of a blob is
the sha256sum of the blob file.

 > Compare: In Git, each piece of content is called an "object".  The
 > content of an object file is in pseudo-code)
 >
 >     zlib.compress(sprintf("%s %d\0%s", obj_type, len(obj_content), obj_content))
 >
 > The ID of an object is the sha1sum of the blob file.
 >
 > In the above psuedo-code, the `obj_type` is the type of content
 > being stored in the object; this information is encoded in to the
 > object storage.  In Docker/OCI, this information is not encoded
 > anywhere; if you have a blob and don't know ahead of time what type
 > of content you're expecting then you'll just have to sniff the
 > content itself and make a guess.

There are several types of data that are encoded in to a blob:

 - "layer": A possibly compressed tarball of a filesystem layer.

 - "config" AKA "image": A JSON file that contains image metadata and
   a list the DiffIDs of layers that make up the image filesystem.
   The blob ID of a config (the sha256sum of the config file) is used
   as the image ID.

 - "manifest": A JSON file describing a config and how to aquire the
   blobs of the layers that it refers to.

 - "index" AKA "manifest list" AKA "fat manifest": A JSON file that is
   a list of blob IDs of manifests.  This may be used in place of a
   single manifest when there is a choice of which image is actually
   desired (you might call it a "fat manifest" in this situation) (as
   for "multi-platform images"; which are really just a separate image
   for each platform), or when you simply need to represent a listing
   of unrelated images (you wouldn't call this a "fat manifest").

 > Compare: In Git, object type names have short simple names: "blob"
 > (regular file), "tree" (directory), "commit", "tag", and maybe a
 > few others.
 >
 > The words in quotes above ("layer", "config", "manifest", "index")
 > are the words that *humans* in the Docker/OCI world use; for
 > machine use there are long unwieldy MIME content-types.  (But they
 > all start with `application/vnd.`, so what's the point?)

If you're like me, at this point you have a burning question: What's
the point of having the manifest separate from the config?  Well, this
question turns out to be a bit of a rabbit-hole, each answer only
raising more questions.

 - Q: What's the point of having the manifest separate from the config;
   it just transcludes info from the config, and contains fancy MIME-y
   references to the layer blobs rather than the config's simple list
   of IDs.

   A: Well, if you were reading carefully, then you may have caught
   that the config contains a list of layer *DiffIDs*, which are
   different than *blob IDs*.  The *DiffID* in a config is the
   sha256sum of the *uncompressed* layer tarball.  The blob might be
   compressed, in which case the *blob ID* would be of the compressed
   tarball, and the two IDs won't be the same!  So with just the
   config's DiffIDs you wouldn't be able to retrieve the layer blobs
   from the storage system.

 - Q: So why not just agree to use the same ID in both places (define
   the blob ID and the diff ID to be the same), totally removing the
   need for the manifest to be separate from the config?

   A: Because the Docker "distribution" folks and the Docker "image"
   folks were apparently incapable of talking with or coordinating
   with eachother.

 - Q: Why not just put both the blob ID and the DiffID in the config?

   A: Again, cross-team coordination.  But also because it may be
   desirable to compress a layer after-the-fact (giving it a new blob
   ID), and that shouldn't change the image ID.  Unlike most blob IDs,
   image IDs are a user-facing thing and so care should be taken that
   they aren't brittle.  For instance, having a registry may wish to
   have a background job that goes through and compresses layer files,
   to save on storage and bandwidth.

 - Q: Are you telling me that there are multiple ways of storing to
   the same content?

   A: Yes, there are.

 - Q: Doesn't that miss the point of content-addressable storage?

   A: Yes, it does.

 - Q: Shouldn't such compression either be built-in to the
   content-addressable system (as in Git), or be a property of the
   transfer and not the blob itself (as in HTTP)?

   A: Yes, that would address all of these problems.  But neither the
   Docker folks nor the OCI folks decided to do that.  One of many
   questionable decisions.

## File format details

### Layers

There are several content-type strings for layer files:

  | Media-Type                                                     | Specified by                                                                       |
  |----------------------------------------------------------------|------------------------------------------------------------------------------------|
  | `application/vnd.docker.image.rootfs.diff.tar.gzip`            | [Docker Image][] (unzipped content), [Docker Manifest v2, Schema 2][] (Media-Type) |
  | `application/vnd.docker.image.rootfs.foreign.diff.tar.gzip`    | [Docker Image][] (unzipped content), [Docker Manifest v2, Schema 2][] (Media-Type) |
  | `application/vnd.oci.image.layer.v1.tar`                       | [OCI Image Format: Filesystem Layers][]                                            |
  | `application/vnd.oci.image.layer.v1.tar+gzip`                  | [OCI Image Format: Filesystem Layers][]                                            |
  | `application/vnd.oci.image.layer.v1.tar+zstd`                  | [OCI Image Format: Filesystem Layers][]                                            |
  | `application/vnd.oci.image.layer.nondistributable.v1.tar`      | [OCI Image Format: Filesystem Layers][]                                            |
  | `application/vnd.oci.image.layer.nondistributable.v1.tar+gzip` | [OCI Image Format: Filesystem Layers][]                                            |
  | `application/vnd.oci.image.layer.nondistributable.v1.tar+zstd` | [OCI Image Format: Filesystem Layers][]                                            |

So, a possibly compressed tarball, but what's in the tarball?  A layer
as would be used with a union/overlay filesystem/mount system;
handling deletions using aufs-style whiteout files.  aufs was Another
Union File System for the Linux kernel that was worked on 2006-2018
but ultimately got rejected from the Linux kernel.  But even though it
got rejected, here in Docker/OCI land we're still dealing with its
legacy!

Anyway, these are pretty straight-forard.  `ocibuild` (by virtue of
using the [google/go-containerregistry][] library) can sniff the
compression and open layers in any of the formats; but `ocibuild` only
emits non-compressed layers (not because we object to compressed
layers, just because we're lazy and haven't particularly wanted them).
`ocibuild layer squash` does roll its own whiteout support, since
[google/go-containerregistry][]'s whiteout support is incomplete (and
also the shape of the library doesn't suit ocibuild's needs; see the
comments on `./pkg/squash.Squash`).

### Configs

There are several content-type strings for config files:

  | Media-Type                                       | Specified by                                                                                           |
  |--------------------------------------------------|--------------------------------------------------------------------------------------------------------|
  | `application/vnd.docker.container.image.v1+json` | [Docker Image][] (content; referred to as "image JSON"), [Docker Manifest v2, Schema 2][] (Media-Type) |
  | `application/vnd.oci.image.config.v1+json`       | [OCI Image Format: Image Configuration][]                                                              |

Docker stores configs at
`/var/lib/docker/image/btrfs/imagedb/content/sha256/{hash}`.

### Manifests

There are several content-type strings for manifest files:

  | Media-Type                                                  | Specified by                         | Notes                                   |
  |-------------------------------------------------------------|--------------------------------------|-----------------------------------------|
  | (no name; entries in `manifest.json` file)                  | [Docker Image v1.1.0+][Docker Image] | Introduced in Docker 1.10.0, 2016-02-04 |
  | `application/vnd.docker.distribution.manifest.v1+json`      | [Docker Manifest v2, Schema 1][]     | Introduced in Docker 1.3.0, 2014-10-15  |
  | `application/vnd.docker.distribution.manifest.v1+prettyjws` | [Docker Manifest v2, Schema 1][]     | Introduced in Docker 1.3.0, 2014-10-15  |
  | `application/vnd.docker.distribution.manifest.v2+json`      | [Docker Manifest v2, Schema 2][]     | Introduced in Docker 1.10.0, 2016-02-04 |
  | `application/vnd.oci.image.manifest.v1+json`                | [OCI Image Format: Image Manifest][] |                                         |

### Indexes (manifest lists)

There are several content-type strings for index files:

  | Media-Type                                                  | Specified by                         | Notes                                   |
  |-------------------------------------------------------------|--------------------------------------|-----------------------------------------|
  | (no name; `repositories` file)                              | [Docker Image][]                     |                                         |
  | (no name; `manifest.json` file)                             | [Docker Image v1.1.0+][Docker Image] | Introduced in Docker 1.10.0, 2016-02-04 |
  | `application/vnd.docker.distribution.manifest.list.v2+json` | [Docker Manifest v2, Schema 2][]     | Introduced in Docker 1.10.0, 2016-02-04 |
  | `application/vnd.oci.image.index.v1+json`                   | [OCI Image Format: Image Index][]    |                                         |

## Bundles

OK, now that we have an idea of the different files that go in to a
complete image, how do we bundle them all up in to a single file for
easy use?

Well, several groups of folks have come up with different answers to
that.

### `docker load` v1

This is for the old v1 data model, so we won't discuss it too much, but

It it's a tarball that looks like

    ├── repositories
    ├── {layer_id}
    │   ├── VERSION
    │   ├── json
    │   └── layer.tar
    └── {layer_id}
        ├── VERSION
        ├── json
        └── layer.tar

Layer IDs were arbitrary 64-character hex names.

Since there was no separate manifest in this data model, the
`repositories` index is pretty simple, something like

   ```json
   {
       "repo_name":{
           "tag_name":"{top_layer_id}"
       }
   }
   ```

This is specified by [Docker Image v1.0.0][].  This is implemented by
`github.com/docker/docker/image/tarexport.tarexporter.Load()`→`.legacyLoad()`.

### `docker load` v2

This was adopted in Docker 1.10.0 (2016-02-04), as part of the
transition to the v2 data model.

This is a little more unstructured.  The only filename that is
actually cared about is `manifest.json` (which would perhaps more
accurately be named `manifests.json`; it contains a list of
manifests).

Here's an example `manifest.json`

   ```json
   [
     {
       "Config": "busybox-config.json",
       "RepoTags": ["busybox:latest"],
       "Layers": [
         "busybox-layer-1.tar.gz",
         "busybox-layer-2.tar.gz"
       ]
     }
   ]
   ```

It's mostly just a list of filenames!  What those filenames are
doesn't matter.

As far as I can tell, there's no spec for this exactly, but it's
more-or-less "[Docker Image v1.1.0+][Docker Image v1.1.0], but without
all of of the backward-compatibility stuff."  It is implemented by the
main `github.com/docker/docker/image/tarexport.tarexporter.Load()`
codepath, and by
`github.com/google/go-containerregistry/pkg/v1/tarball`.

This is a terrible, horrible, no good, very bad format; because of
`manifest.json`:
 - TL;DR: It can't do multi-arch images.
 - ----
 - "But surely badness is the doing of committees; surely it's the
   standardized OCI format!"  Nope!  It's Docker's own format; the OCI
   standardization effort for manifests wouldn't start until
   2016-04-07, 2 months after Docker shipped this.
 - "OK, so surely the bad is because of contstraints from the Docker
   Manifest format, which was network-oriented in design?"  Nope, it's
   not using the Docker Manifest v2 format.
 - "Wait, wouldn't it have made a lot of sense to use the Docker
   Manifest v2 schema 2 format, since that was introduced in the same
   Docker version and (compared to schema 1) introduced URL hints that
   would allow it to point at filenames, the key thing that
   `manifest.json` needs to do that other manifests might not need to
   do?  It seems like all of the other details are pointing to that!"
   Yep, that'd have made lots of sense, except for the
   already-established poor coordination between the Docker
   "distribution" and Docker "image" folks.
 - "So what, is it using the ancient (pre Docker 1.3.0 / 2014-10-15)
   Docker Manifest v1 format?"  Honestly, maybe, I can't find a
   description of what Docker Manifest v1 looked like.  I'm pretty
   sure it's just a custom little thing that ignored any of the other
   manifest work.
 - "Fine, they made silly decisions, but what about it is actually
   bad?"  It can't represent multi-platform images.  Which, by the
   way, support was added for in Docker 1.10.0--the same version when
   they made up this format.  Why couldn't the Docker folks coordinate
   with eachother within their org!?

### `docker save` (combination of `docker load` v1 and v2)

This is "let's craft a tarball that is both valid under the old format
and the new format"; this is possible by including both a
`repositories` and a `manifest.json` and naming the layer tarballs
`{layer_id}/layer.tar`.

This does mean that there's the restriction that layers can't be
compressed.

This is specified by [Docker Image v1.1.0+][Docker Image v1.1.0].
Creating this type of file is implemented by
`github.com/docker/docker/image/tarexport.tarexporter.Save()`, and by
`github.com/google/go-containerregistry/pkg/legacy/tarball`.

Why does google/go-containerregistry have this as a separate
`legacy/tarball` package instead of maximizing the compatibility of
`v1/tarball` output?  I assume because `v1/tarball` really wants to be
able to compress layers.  IDK.  Also, I suspect that the implementor
of `legacy/tarball` (which was added way after `v1/tarball`) was
confused about what the `v1` in `v1/tarball` was referring to (heck,
I'm confused what it refers to).

### OCI layout

The [OCI Image spec][OCI Image Format: Image Layout] defines a way to
lay out the files in a directory structure, and notes that it may be
used in a tarball.

 - There's a `oci-layout` file that identifies the directory/tarball
   as following this format.

 - There's an `index.json` file that is an index of all of the images
   contained in the directory/tarball.  Per the usual OCI
   content-addressable storage things, this points at manifests (or
   fat manifests) by their blob ID.

 - Everything else are files at `blobs/sha256/{blob_id}`.

Because this is the standardized vendor-neutral format from the
working group launched by Docker, most tools support this format.
Except for Docker itself (as of Docker 20.10.14).  `docker save` can't
emit this, and `docker load` can't load it.  There was a PR filed in
2017, but it never got merged.
:smiling-face-but-you-can-see-pain-in-their-eyes:

This is specified by [OCI Image Format: Image Layout][].  This is
implemented by , but without all of of the backward-compatibility
stuff."  This is implemented by
`github.com/google/go-containerregistry/pkg/v1/layout`, but weirdly
that package only deals with directories on a filesystem; and can't
work with tarballs.  Same with the competing
`github.com/containers/image` library.  The `skopeo` implements this,
and is from the same github.com/containers folks, but does so by using
`github.com/containers/image` and a temporary directory.

### What `ocibuild` does

Originally, `ocibuild` worked with images as the `docker load` v2
format, because that's what
`github.com/google/go-containerregistry/pkg/v1/tarball` does, and at
the time I was under the mistaken impression that it was a
standardized OCI format.  Also because it made running images with
`docker load` convenient.

But as we contemplate multi-arch support, we may want to switch to OCI
layout tarballs; and either require a 3rd-party tool to load them in
to dockerd (such as `skopeo`) or add such push-to-dockerd
functionality to `ocibuild`.

[google/go-containerregistry]: ./go-containerregistry.md

[Docker Image]:        https://github.com/moby/moby/blob/master/image/spec
[Docker Image v1.0.0]: https://github.com/moby/moby/blob/master/image/spec/v1.md
[Docker Image v1.1.0]: https://github.com/moby/moby/blob/master/image/spec/v1.1.md
[Docker Image v1.2.0]: https://github.com/moby/moby/blob/master/image/spec/v1.2.md

[Docker Manifest v2, Schema 1]: https://github.com/distribution/distribution/blob/main/docs/spec/manifest-v2-1.md
[Docker Manifest v2, Schema 2]: https://github.com/distribution/distribution/blob/main/docs/spec/manifest-v2-2.md
[Docker Registry HTTP API v2]:  https://github.com/distribution/distribution/blob/main/docs/spec/api.md

[OCI Image Format: Filesystem Layers]:   https://github.com/opencontainers/image-spec/blob/main/layer.md
[OCI Image Format: Image Configuration]: https://github.com/opencontainers/image-spec/blob/main/config.md
[OCI Image Format: Image Manifest]:      https://github.com/opencontainers/image-spec/blob/main/manifest.md
[OCI Image Format: Image Index]:         https://github.com/opencontainers/image-spec/blob/main/image-index.md
[OCI Image Format: Image Layout]:        https://github.com/opencontainers/image-spec/blob/main/image-layout.md
