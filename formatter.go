package main

// Formatter defines the format method that receives the value it should operate on and the current
// renamer state. All formatter must adhere to this interface to be part of a FormattingPipeline
type Formatter interface {
	// Format receives a property value and the state of the renamer, applies the formatting
	// implementation and returns the formatted string. The value is passed to the next formatter
	// in the pipeline or inserted in the output string
	Format(string, renamerState) (string, error)
}

// PaddingFormatter can pad a value with another character. For example, it can be used to zero-pad
// the counter: $cnt[%03]
type PaddingFormatter struct {
	PadCharacter rune
	PadLength    int
}

// NewPaddingFormatter creates a new padding formatter that pads a property with the given char up
// to the maximum length
func NewPaddingFormatter(char rune, length int) PaddingFormatter {
	return PaddingFormatter{
		PadCharacter: char,
		PadLength:    length,
	}
}

// Format applies padding to the given value with the character configured
func (f *PaddingFormatter) Format(value string, rstate renamerState) (string, error) {
	if len(value) >= f.PadLength {
		return value, nil
	}

	newValue := ""
	idx := 0
	for idx < f.PadLength-len(value) {
		newValue += string(f.PadCharacter)
		idx++
	}
	newValue += value
	return newValue, nil
}

// SliceFormatter can cut a substring out of a value. The slice formatter is called with the &gt;
// character and receives two parameters separated by ":": The beginning index at which to cut the value
// and the end index. If either value is left blank the system assumes 0 for the beginning and max
// for the end. For example, >:10 declares a formatter that cuts values to a maximum length of 10
// characters. If the starting index is higher than the length of the value the formatter will return
// an empty string.
type SliceFormatter struct {
	Start int
	End   int
}

// NewSliceFormatter initializes a new SliceFormatter with the given start and end indexes
func NewSliceFormatter(start, end int) SliceFormatter {
	return SliceFormatter{
		Start: start,
		End:   end,
	}
}

// Format trims the string to the start and end points specified by the SliceFormatter
func (f *SliceFormatter) Format(value string, rstate renamerState) (string, error) {
	strlen := len(value)
	strstart := 0
	if f.Start > 0 {
		strstart = f.Start
	}
	if strstart > strlen {
		return "", nil
	}
	if f.End > 0 && f.End < strlen {
		strlen = f.End
	}
	return value[strstart:strlen], nil
}
