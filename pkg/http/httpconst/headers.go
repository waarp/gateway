// Package httpconst regroups a list of constants defining the various headers
// and URL variables which can be used when making HTTP transfers with the
// gateway, to activate certain features.
package httpconst

const (
	// TransferID defines the name of the (optional) request HTTP header
	// containing the ID of the requested transfer.
	TransferID = "Waarp-Transfer-ID"

	// RuleName defines the name of the (optional) request HTTP header
	// containing the name of the rule to be used during the transfer.
	RuleName = "Waarp-Rule-Name"

	// TransferStatus defines the name of the response HTTP header containing
	// the status in which the transfer ended.
	TransferStatus = "Waarp-Transfer-Status"

	// ErrorCode defines the name of the response HTTP header containing the
	// error code, if an error occurred.
	ErrorCode = "Waarp-Error-Code"

	// ErrorMessage defines the name of the response HTTP header containing the
	// error message, if an error occurred.
	ErrorMessage = "Waarp-Error-Message"
)

const (
	// TransferInfo defines the name of the header containing all the transfer
	// info. The information should be presented as a list of key:value pairs.
	// A transfer info can only be transmitted by the client to the server, never
	// the other way around.
	TransferInfo = "Waarp-Transfer-Info"

	/*
		// FileInfo defines the name of the header containing all the file
		// info. The information should be presented as a list of key:value pairs.
		// A file info can only be transmitted by the sender to the receiver, never
		// the other way around.
		FileInfo = "Waarp-File-Info"
	*/
)
