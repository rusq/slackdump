# Troubleshooting Common Issues

## ZIP file extraction error on macOS (Illegal byte sequence)

__Symptoms:__

```
$ uzip slackdump_20250101_150607.zip

error:  cannot create __uploads/F08CHQGJL4A/Screenshot 2025-01-22 at 9.10.05ǻAM.txt
        Illegal byte sequence
```

__Solution:__

Some filenames have UTF-8 characters that are not supported by "unzip" command
on macOS.  Luckily, you can use the `tar` command to extract the ZIP file (or
`bsdtar`):

```bash
$ tar xf slackdump_20250101_150607.zip
```

If the `tar` or `bsdtar` is not available, you can use slackdump's unzip tool:

```bash
$ slackdump tools unzip slackdump_20250101_150607.zip
```

For more details, read issue #435 on GitHub.
