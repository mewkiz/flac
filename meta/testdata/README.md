New BSD License
---------------

The following testcase sounds have been copied from the [reference implementation][] library, which is licensed under the New [BSD License][].

* input-SCPAP.flac
* input-SCVA.flac
* input-SCVAUP.flac
* input-SCVPAP.flac
* input-SVAUP.flac
* input-VA.flac

[reference implementation]: https://git.xiph.org/?p=flac.git
[BSD License]: https://git.xiph.org/?p=flac.git;a=blob_plain;f=COPYING.Xiph

public domain
-------------

The following images and sounds have been released into the *[public domain][]*.

* [silence.jpg][]
* silence.flac: Created using the following commands:

```shell
ffmpeg -f lavfi -i "aevalsrc=0|0:d=3" silence.wav
flac silence.wav
metaflac --import-picture=silence.jpg silence.flac
```

[public domain]: https://creativecommons.org/publicdomain/zero/1.0/
[silence.jpg]: http://www.pdpics.com/photo/2546-silence-word-magnified/
