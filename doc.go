package main

const cliDescription = "raf renames multiple files or directories in a single pass. The name for the new files is generated based " +
	"on a set of properties extracted from the original file name using the -p option, intrinsic variables such as $cnt for a counter, " +
	"and literal strings. raf stores a record of the changes it made to a .raf file in the folder containing the files to be renamed " +
	"and, using this record, can undo its changes with the undo command.\n\nBy default all logs are sent to stderr and the new file " +
	"names are printed to stdout as files are renamed. The format of the output changes when executing in dry run mode - see dry run flag."

const propFlagDescription = "The prop flag tells raf to extract text from the original file name and store it in a variable " +
	"that can be used to generate the new file name. A prop flag is assigned a unique name and a regular expression used to extract " +
	"the value from the original file name: -p \"title=Video\\ \\d+\\ \\-\\ ([A-Za-z0-9\\ ]+)_\". If the regular expression collects " +
	"a group using () only the content of the group is assigned to the variable, otherwise raf collects the entire match."

const outputFlagDescription = "The output flag specifies the pattern to generate new file names. To include a variable it should be " +
	"referenced with a dollar ($) sign. Additionally, variable can include formatting directives enclosed in square brackets [ ] after their " +
	"name. Raf provides the following variables: $cnt - counter of files processed starting at 1; $ext - extension of the original file; " +
	"$fname - full name of the original file excluding its extension."

const dryRunFlagDescription = "Dry run mode makes raf print to stderr the operations it would perform in the format \"File <original name> " +
	"-> <new file name>\" without actually renaming the file."
