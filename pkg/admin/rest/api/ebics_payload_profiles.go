package api

type InEbicsPayloadProfile struct {
	Name                   string         `json:"name" yaml:"name"`
	Label                  string         `json:"label,omitempty" yaml:"label,omitempty"`
	Description            string         `json:"description,omitempty" yaml:"description,omitempty"`
	OrderType              string         `json:"orderType" yaml:"orderType"`
	Direction              string         `json:"direction" yaml:"direction"`
	ServiceName            string         `json:"serviceName,omitempty" yaml:"serviceName,omitempty"`
	ServiceOption          string         `json:"serviceOption,omitempty" yaml:"serviceOption,omitempty"`
	Scope                  string         `json:"scope,omitempty" yaml:"scope,omitempty"`
	MsgName                string         `json:"msgName,omitempty" yaml:"msgName,omitempty"`
	ContainerType          string         `json:"containerType,omitempty" yaml:"containerType,omitempty"`
	DefaultRule            string         `json:"defaultRule,omitempty" yaml:"defaultRule,omitempty"`
	DefaultTargetDirectory string         `json:"defaultTargetDirectory,omitempty" yaml:"defaultTargetDirectory,omitempty"`
	RequiresDeclaredAmount bool           `json:"requiresDeclaredAmount,omitempty" yaml:"requiresDeclaredAmount,omitempty"`
	DefaultCurrency        string         `json:"defaultCurrency,omitempty" yaml:"defaultCurrency,omitempty"`
	AllowedExtensions      []string       `json:"allowedExtensions,omitempty" yaml:"allowedExtensions,omitempty"`
	FilenamePattern        string         `json:"filenamePattern,omitempty" yaml:"filenamePattern,omitempty"`
	StrictContractCheck    *bool          `json:"strictContractCheck,omitempty" yaml:"strictContractCheck,omitempty"`
	IsEnabled              *bool          `json:"isEnabled,omitempty" yaml:"isEnabled,omitempty"`
	Metadata               map[string]any `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

type OutEbicsPayloadProfile struct {
	ID                     int64          `json:"id" yaml:"id"`
	Name                   string         `json:"name" yaml:"name"`
	Label                  string         `json:"label,omitempty" yaml:"label,omitempty"`
	Description            string         `json:"description,omitempty" yaml:"description,omitempty"`
	OrderType              string         `json:"orderType" yaml:"orderType"`
	Direction              string         `json:"direction" yaml:"direction"`
	ServiceName            string         `json:"serviceName,omitempty" yaml:"serviceName,omitempty"`
	ServiceOption          string         `json:"serviceOption,omitempty" yaml:"serviceOption,omitempty"`
	Scope                  string         `json:"scope,omitempty" yaml:"scope,omitempty"`
	MsgName                string         `json:"msgName,omitempty" yaml:"msgName,omitempty"`
	ContainerType          string         `json:"containerType,omitempty" yaml:"containerType,omitempty"`
	DefaultRule            string         `json:"defaultRule,omitempty" yaml:"defaultRule,omitempty"`
	DefaultTargetDirectory string         `json:"defaultTargetDirectory,omitempty" yaml:"defaultTargetDirectory,omitempty"`
	RequiresDeclaredAmount bool           `json:"requiresDeclaredAmount" yaml:"requiresDeclaredAmount"`
	DefaultCurrency        string         `json:"defaultCurrency,omitempty" yaml:"defaultCurrency,omitempty"`
	AllowedExtensions      []string       `json:"allowedExtensions,omitempty" yaml:"allowedExtensions,omitempty"`
	FilenamePattern        string         `json:"filenamePattern,omitempty" yaml:"filenamePattern,omitempty"`
	StrictContractCheck    bool           `json:"strictContractCheck" yaml:"strictContractCheck"`
	IsEnabled              bool           `json:"isEnabled" yaml:"isEnabled"`
	Metadata               map[string]any `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}
