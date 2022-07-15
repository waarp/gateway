package rest

func setIfDefined(in, out *string) {
	if in != nil {
		*out = *in
	}
}
