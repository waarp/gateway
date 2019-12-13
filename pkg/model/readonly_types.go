package model

// ReadOnlyByte is an alias for the `byte` type. All struct fields with this
// type will not be unmarshalled from a JSON, which means they cannot be modified
// (hence the "read-only" name).
type ReadOnlyByte byte

// UnmarshalJSON is the method called when unmarshalling a `ReadOnlyByte` var
// from a JSON. This method ignores the JSON data and leaves the var empty.
func (ReadOnlyByte) UnmarshalJSON(data []byte) error {
	return nil
}

// ReadOnlyString is an alias for the `string` type. All struct fields with this
// type will not be unmarshalled from JSON, which means they cannot be modified
// (hence the "read-only" name).
type ReadOnlyString string

// UnmarshalJSON is the method called when unmarshalling a `ReadOnlyString` var
// from a JSON. This method ignores the JSON data and leaves the var empty.
func (ReadOnlyString) UnmarshalJSON(data []byte) error {
	return nil
}
