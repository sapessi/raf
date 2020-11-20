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
