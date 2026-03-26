package wg

import (
	"fmt"
	"io"
	"path"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

func buildPayloadRequest(
	commandProfile, commandRule, hostID, partnerID, userID,
	filePath, outputName, targetDirectory,
	orderType, serviceName, serviceOption, scope, msgName, containerType string,
	metadata map[string]confVal,
) *api.InEbicsPayloadRequest {
	request := &api.InEbicsPayloadRequest{
		Profile: commandProfile,
		Rule:    commandRule,
		Subscriber: api.InSubscriberRef{
			HostID:    hostID,
			PartnerID: partnerID,
			UserID:    userID,
		},
	}

	if strings.TrimSpace(filePath) != "" || strings.TrimSpace(outputName) != "" {
		request.File = &api.InPayloadFile{
			Path:       filePath,
			OutputName: outputName,
		}
	}

	if strings.TrimSpace(targetDirectory) != "" {
		request.Target = &api.InPayloadTarget{Directory: targetDirectory}
	}

	request.Service = &api.InPayloadService{
		OrderType:     orderType,
		ServiceName:   serviceName,
		ServiceOption: serviceOption,
		Scope:         scope,
		MsgName:       msgName,
		ContainerType: containerType,
	}

	if metadata != nil {
		request.Metadata = map[string]any{}
		for key, value := range metadata {
			request.Metadata[key] = value
		}
	}

	return request
}

//nolint:lll // CLI tags are intentionally explicit
type EbicsPayloadUpload struct {
	OrderType     string             `required:"yes" long:"order-type" choice:"BTU" choice:"FUL" description:"The EBICS payload order type"`
	Profile       string             `long:"profile" description:"The EBICS payload profile name"`
	Rule          string             `long:"rule" description:"The Gateway transfer rule name"`
	HostID        string             `required:"yes" long:"host-id" description:"The EBICS host ID"`
	PartnerID     string             `required:"yes" long:"partner-id" description:"The EBICS partner ID"`
	UserID        string             `required:"yes" long:"user-id" description:"The EBICS user ID"`
	File          string             `required:"yes" long:"file" description:"The source payload file path"`
	OutputName    string             `long:"output-name" description:"The remote payload name"`
	ServiceName   string             `long:"service-name" description:"The EBICS service name"`
	ServiceOption string             `long:"service-option" description:"The EBICS service option"`
	Scope         string             `long:"scope" description:"The EBICS scope"`
	MsgName       string             `long:"msg-name" description:"The EBICS message name"`
	ContainerType string             `long:"container-type" description:"The EBICS container type"`
	Metadata      map[string]confVal `long:"metadata" description:"Payload metadata in key:value format. Can be repeated."`
}

func (c *EbicsPayloadUpload) Execute([]string) error { return execute(c) }
func (c *EbicsPayloadUpload) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/payloads", strings.ToLower(c.OrderType), "upload")

	request := buildPayloadRequest(
		c.Profile,
		c.Rule,
		c.HostID,
		c.PartnerID,
		c.UserID,
		c.File,
		c.OutputName,
		"",
		c.OrderType,
		c.ServiceName,
		c.ServiceOption,
		c.Scope,
		c.MsgName,
		c.ContainerType,
		c.Metadata,
	)

	if _, err := add(w, request); err != nil {
		return err
	}

	fmt.Fprintf(w, "The EBICS payload %q was successfully submitted.\n", c.File)

	return nil
}

//nolint:lll // CLI tags are intentionally explicit
type EbicsPayloadDownload struct {
	OrderType     string             `required:"yes" long:"order-type" choice:"BTD" choice:"FDL" description:"The EBICS payload order type"`
	Profile       string             `long:"profile" description:"The EBICS payload profile name"`
	Rule          string             `long:"rule" description:"The Gateway transfer rule name"`
	HostID        string             `required:"yes" long:"host-id" description:"The EBICS host ID"`
	PartnerID     string             `required:"yes" long:"partner-id" description:"The EBICS partner ID"`
	UserID        string             `required:"yes" long:"user-id" description:"The EBICS user ID"`
	TargetDir     string             `long:"target-dir" description:"The target directory for downloaded payloads"`
	ServiceName   string             `long:"service-name" description:"The EBICS service name"`
	ServiceOption string             `long:"service-option" description:"The EBICS service option"`
	Scope         string             `long:"scope" description:"The EBICS scope"`
	MsgName       string             `long:"msg-name" description:"The EBICS message name"`
	ContainerType string             `long:"container-type" description:"The EBICS container type"`
	Metadata      map[string]confVal `long:"metadata" description:"Payload metadata in key:value format. Can be repeated."`
}

func (c *EbicsPayloadDownload) Execute([]string) error { return execute(c) }
func (c *EbicsPayloadDownload) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/payloads", strings.ToLower(c.OrderType), "download")

	request := buildPayloadRequest(
		c.Profile,
		c.Rule,
		c.HostID,
		c.PartnerID,
		c.UserID,
		"",
		"",
		c.TargetDir,
		c.OrderType,
		c.ServiceName,
		c.ServiceOption,
		c.Scope,
		c.MsgName,
		c.ContainerType,
		c.Metadata,
	)

	if _, err := add(w, request); err != nil {
		return err
	}

	fmt.Fprintf(w, "The EBICS payload download %q was successfully submitted.\n", c.OrderType)

	return nil
}

type EbicsPayloadList struct {
	ListOptions
}

func (c *EbicsPayloadList) Execute([]string) error { return execute(c) }
func (c *EbicsPayloadList) execute(w io.Writer) error {
	addr.Path = "/api/ebics/payloads"
	listURL(&c.ListOptions, "")

	var body map[string][]*api.OutEbicsOperation
	if err := list(&body); err != nil {
		return err
	}

	if payloads := body["payloads"]; len(payloads) > 0 {
		return outputObject(w, payloads, &c.OutputFormat, displayEbicsOperations)
	}

	fmt.Fprintln(w, "No EBICS payload found.")

	return nil
}

type EbicsPayloadGet struct {
	OutputFormat

	Args struct {
		Operation string `required:"yes" positional-arg-name:"operation" description:"The payload operation identifier"`
	} `positional-args:"yes"`
}

func (c *EbicsPayloadGet) Execute([]string) error { return execute(c) }
func (c *EbicsPayloadGet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/payloads", c.Args.Operation)

	var operation api.OutEbicsOperation
	if err := get(&operation); err != nil {
		return err
	}

	return outputObject(w, &operation, &c.OutputFormat, displayEbicsOperation)
}

type EbicsPayloadRetry struct {
	Args struct {
		Operation string `required:"yes" positional-arg-name:"operation" description:"The payload operation identifier"`
	} `positional-args:"yes"`

	Reason   string             `long:"reason" description:"Operator reason for the retry" json:"reason,omitempty"`
	Metadata map[string]confVal `long:"metadata" description:"Operator metadata." json:"metadata,omitempty"`
}

func (c *EbicsPayloadRetry) Execute([]string) error { return execute(c) }
func (c *EbicsPayloadRetry) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/payloads", c.Args.Operation, "retry")

	if err := updateMethod(w, c, "PUT"); err != nil {
		return err
	}

	fmt.Fprintf(w, "The EBICS payload %q was successfully scheduled for retry.\n", c.Args.Operation)

	return nil
}

type EbicsPayloadRecover struct {
	Args struct {
		Operation string `required:"yes" positional-arg-name:"operation" description:"The payload operation identifier"`
	} `positional-args:"yes"`

	Reason   string             `long:"reason" description:"Operator reason for the recovery" json:"reason,omitempty"`
	Metadata map[string]confVal `long:"metadata" description:"Operator metadata." json:"metadata,omitempty"`
}

func (c *EbicsPayloadRecover) Execute([]string) error { return execute(c) }
func (c *EbicsPayloadRecover) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/payloads", c.Args.Operation, "recover")

	if err := updateMethod(w, c, "PUT"); err != nil {
		return err
	}

	fmt.Fprintf(w, "The EBICS payload %q was successfully scheduled for recovery.\n", c.Args.Operation)

	return nil
}
