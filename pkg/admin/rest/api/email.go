package api

type GetEmailTemplateObject struct {
	Name        string   `json:"name"`
	Subject     string   `json:"subject"`
	MIMEType    string   `json:"mimeType"`
	Body        string   `json:"body"`
	Attachments []string `json:"attachments"`
}

type PostEmailTemplateObject struct {
	Name        string   `json:"name"`
	Subject     string   `json:"subject"`
	MIMEType    string   `json:"mimeType"`
	Body        string   `json:"body"`
	Attachments []string `json:"attachments"`
}

type PatchEmailTemplateObject struct {
	Name        Nullable[string] `json:"name,omitempty"`
	Subject     Nullable[string] `json:"subject,omitempty"`
	MIMEType    Nullable[string] `json:"mimeType,omitempty"`
	Body        Nullable[string] `json:"body,omitempty"`
	Attachments []string         `json:"attachments,omitempty"`
}

type GetSMTPCredentialObject struct {
	EmailAddress  string `json:"emailAddress"`
	ServerAddress string `json:"serverAddress"`
	Login         string `json:"login"`
	Password      string `json:"password"`
}

type PostSMTPCredentialObject struct {
	EmailAddress  string `json:"emailAddress"`
	ServerAddress string `json:"serverAddress"`
	Login         string `json:"login"`
	Password      string `json:"password"`
}

type PatchSMTPCredentialObject struct {
	EmailAddress  Nullable[string] `json:"emailAddress"`
	ServerAddress Nullable[string] `json:"serverAddress"`
	Login         Nullable[string] `json:"login"`
	Password      Nullable[string] `json:"password"`
}
