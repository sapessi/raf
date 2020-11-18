# Rename All Files (raf) 

`raf` makes it easy to rename multiple files all at once. The name for the new files is constructed from parts of the original file name and can include additional string literals or intrinsic variables such as counters. Renaming multiple files in a folder is a delicate operation so `raf` allows both execution in dry-run mode (`-d`) that simply prints out the changes it would make and rollback by creating a `.raf` file in the folder that contains the orioginl file names.

![](media/raf.gif)


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

## stdout, stderr
`raf` sends all log output to stderr. The stdout only receives the new file names separate by `\n`. This makes it easy to use it in combination with other commands. When executed in dry-run mode the `stdout` is: `File <original file name> -> <new file name>`

## TODO
- [ ] tests tests tests
- [ ] Output formatting, for example $cnt[%03d] to obtain `001`