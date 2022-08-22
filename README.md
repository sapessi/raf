# Rename All Files (raf) 

`raf` makes it easy to rename multiple files all at once. The name for the new files is constructed from parts of the original file name and can include additional string literals or intrinsic variables such as counters. Renaming multiple files in a folder is a delicate operation so `raf` allows both execution in dry-run mode (`-d`) that simply prints out the changes it would make and rollback by creating a `.raf` file in the folder that contains the orioginl file names.

![](media/raf.gif)

For example, you could use `raf` to rename GoPro video files so that they sort correctly when listed alphabetically:
```bash
$ raf -d -p 'chap=G[PHXL](\d{2})' -p 'seq=G[PHXL]\d{2}(\d{4})' -o 'GoPro_$seq_$chap$ext' *
```

## Options:
* `--prop -p`: Specifies a regular expression to select a part of the file name and saves its value in the variable name. `<name>=<regex>`
* `--output -o`: Specifies the format of the output using the variables selected through the `-p` options as well as the default/generated variables
* `--dryrun -d`: Runs the command in dry run mode. When in dry run mode the log output is sent to stderr and the changed file names are sent to stdout, the files are not actually renamed
* `--verbose -v`: Prints verbose log output

## Intrinsic variables
These variables are automatically made available during execution and can be referenced in the output text
* `$cnt`: Counter starting from 1 and incremented for each file
* `$ext`: Extension of the original file
* `$fname`: Full original file name

## Undo
`raf` saves a `.raf` status file in the folder where it was executed. If you run the `raf undo` command `raf` reads the status file and restore the files to their original name.

## Output formatting
Properties in the output support formatters. As of today, only a padding formatter is available. However, `raf`'s code is ready to support a pipeline of different formatters. The padding formatter makes it easy to pad properties with a character. For example, you can use the padding formatter to zero-pad a number in the output. This output string `raf -o 'test - $cnt[%03].mkv' *` will produce the following file name `test - 001.mkv`.

The padding formatter is triggered with the `%` character and receives two parameters. First, a single character (`0` in our example) that should be used for padding. Second, a number that represents the length of the field (`3` in our example). If you we had specified `$cnt[%a5]` the output would have been `aaaa1`.

## stdout, stderr
`raf` sends all log output to stderr. The stdout only receives the new file names separate by `\n`. This makes it easy to use it in combination with other commands. When executed in dry-run mode the `stdout` is: `File <original file name> -> <new file name>`

## TODO
- [ ] tests tests tests
