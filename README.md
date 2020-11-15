# Rename All Files (raf)

This command line utility renames multiple files based on a set of rules. The rules allow you to select portions of the original file name using regular expression and then compose them in a new name. 

```bash
$ ls 
[Wedding]UnionStudio - Video 1 - Home_(10bit_BD720p_x265).mkv
[Wedding]UnionStudio - Video 2 - Venue_(10bit_BD720p_x265).mkv
[Wedding]UnionStudio - Video 3 - Church_(10bit_BD720p_x265).mkv
[Wedding]UnionStudio - Video 4 - Reception_(10bit_BD720p_x265).mkv
[Wedding]UnionStudio - Video 5 - Day2_(10bit_BD720p_x265).mkv

$ raf -p "title=Video\ \d+\ \-\ ([A-Za-z0-9\ ]+)_" -d -o 'UnionStudio - $cnt - $title.mkv' *
```
**Remember to use single quotes for the output name so that the `$` won't be interpreted by the shell**

## Syntax
```bash
$ raf -p var=/selector regex/ -p var2=/selector regex/ -o "output" <files selector>
```

## Options:
* `--prop -p`: Specifies a regular expression to select a part of the file name and saves its value in the variable name. `<title>=/regex/`
* `--output -o`: Specifies the format of the output using the variables selected through the `-p` options as well as the default/generated variables
* `--dryrun -d`: Runs the command in dry run mode. When in dry run mode the log output is sent to stderr and the changed file names are sent to stdout, the files are not actually renamed
* `--verbose -v`: Prints verbose log output

## Default variables
* `$cnt`: Counter starting from 1 and incremented for each file

## TODO
- [x] Default variables `ext` and `fname`
- [ ] Sort parameters
- [ ] tests tests tests
- [ ] Quiet mode options
- [ ] Variable formatting
- [ ] Undo

### Formatting
- zero padded no: $cnt[000]
- upper case: $title[^]
- trim: $title[0:10]
- camel case: $title[_^_]
- pascal case: $title[^_^]
- replace: $title[/-/ /]