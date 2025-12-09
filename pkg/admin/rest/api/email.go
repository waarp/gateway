package api

type GetEmailTemplateObject struct {
	Name        string   `json:"name" yaml:"name"`
	Subject     string   `json:"subject" yaml:"subject"`
	MIMEType    string   `json:"mimeType" yaml:"mimeType"`
	Body        string   `json:"body" yaml:"body"`
	Attachments []string `json:"attachments" yaml:"attachments"`
}

type PostEmailTemplateObject struct {
	Name        string   `json:"name" yaml:"name"`
	Subject     string   `json:"subject" yaml:"subject"`
	MIMEType    string   `json:"mimeType" yaml:"mimeType"`
	Body        string   `json:"body" yaml:"body"`
	Attachments []string `json:"attachments" yaml:"attachments"`
}

type PatchEmailTemplateObject struct {
	Name        Nullable[string] `json:"name,omitzero" yaml:"name,omitempty"`
	Subject     Nullable[string] `json:"subject,omitzero" yaml:"subject,omitempty"`
	MIMEType    Nullable[string] `json:"mimeType,omitzero" yaml:"mimeType,omitempty"`
	Body        Nullable[string] `json:"body,omitzero" yaml:"body,omitempty"`
	Attachments []string         `json:"attachments,omitempty" yaml:"attachments,omitempty"`
}

type GetSMTPCredentialObject struct {
	EmailAddress  string `json:"emailAddress" yaml:"emailAddress"`
	ServerAddress string `json:"serverAddress" yaml:"serverAddress"`
	Login         string `json:"login" yaml:"login"`
	Password      string `json:"password" yaml:"password"`
}

type PostSMTPCredentialObject struct {
	EmailAddress  string `json:"emailAddress" yaml:"emailAddress"`
	ServerAddress string `json:"serverAddress" yaml:"serverAddress"`
	Login         string `json:"login" yaml:"login"`
	Password      string `json:"password" yaml:"password"`
}

type PatchSMTPCredentialObject struct {
	EmailAddress  Nullable[string] `json:"emailAddress" yaml:"emailAddress"`
	ServerAddress Nullable[string] `json:"serverAddress" yaml:"serverAddress"`
	Login         Nullable[string] `json:"login" yaml:"login"`
	Password      Nullable[string] `json:"password" yaml:"password"`
}
