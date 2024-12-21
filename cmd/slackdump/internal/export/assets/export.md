# Command: export

The `export` command saves your Slack workspace as a directory of files.
By default, it exports the entire workspace that your user can access.
You can customize the archive to include specific channels, groups, or
direct messages by providing their URLs or IDs.

The ZIP file it generates is compatible with the Slack Export format with Slackdump specific extensions.

The export file is understood by Slack Import feature with the following
caveat:
- files will not be imported, unless the `export` token is specified.
  Github user @codeallthethingz has created a script that allows you to
  import attachments from the export file.  You can find it
  [here](https://github.com/rusq/slackdump/issues/371)


## Export file structure

```plaintext
/
├── __uploads              : all uploaded files are placed in this dir.
│   └── F02PM6A1AUA        : slack file ID is used as a directory name
|       └── Chevy.jpg      : file attachment
├── everyone               : channel "#everyone"
│   ├── 2022-01-01.json    :   all messages for the 1 Jan 2022.
│   └── 2022-01-04.json    :    "     "      "   "  4 Jan 2022.
├── DM12345678             : Your DMs with Scumbag Steve^
│   └── 2022-01-04.json    :   (you did not have much to discuss —
│                          :    Steve turned out to be a scumbag)
├── channels.json          : all workspace channels information
├── dms.json               : direct message information
└── users.json             : all workspace users information
```

### Channels
The channels are be saved in directories, named after the channel title,
i.e. `#random` would be saved to "random" directory. The directory will
contain a set of JSON files, one per each day.

### Users
User directories will have an "D" prefix, to find out the user name,
check `users.json` file.

### Group Messages
Group messages will have all involved user handles in their name.

## Inclusive and Exclusive Modes

It is possible to **include** or **exclude** channels in/from the Export.

For more details, run `slackdump help syntax`.

## Viewing the Export

To view the export, run `slackdump view <export_file>`.


