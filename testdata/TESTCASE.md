# Testcase Licences

## BSD License

The following testcase sounds have been copied from the [reference implementation] library, which is released under a [BSD license].

* input-SCPAP.flac
* input-SCVA.flac
* input-SCVAUP.flac
* input-SCVPAP.flac
* input-SVAUP.flac
* input-VA.flac
* `missing-value.flac`, created using the following command.

```shell
sed 's/title=/title /' input-SCVA.flac > missing-value.flac
```

[reference implementation]: https://git.xiph.org/?p=flac.git
[BSD license]: https://git.xiph.org/?p=flac.git;a=blob_plain;f=COPYING.Xiph

## Public domain

The following testcase images and sounds have been released into the [public domain].

* [silence.jpg](http://www.pdpics.com/photo/2546-silence-word-magnified/)
* `silence.flac`, created using the following commands.

```shell
ffmpeg -f lavfi -i "aevalsrc=0|0:d=3" silence.wav
flac silence.wav
metaflac --import-picture=silence.jpg silence.flac
```

[public domain]: https://creativecommons.org/publicdomain/zero/1.0/
