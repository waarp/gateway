package wg

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

func displayEbicsKeyLifecycles(w io.Writer, lifecycles []*api.OutEbicsKeyLifecycle) error {
	Style0.PrintV(w, "=== EBICS key lifecycles ===")
	for _, lifecycle := range lifecycles {
		if err := displayEbicsKeyLifecycle(w, lifecycle); err != nil {
			return err
		}
	}

	return nil
}

func displayEbicsKeyLifecycle(w io.Writer, lifecycle *api.OutEbicsKeyLifecycle) error {
	Style1.Printf(w, "EBICS key lifecycle #%d [%s]", lifecycle.ID, lifecycle.Status)
	Style22.PrintL(w, "Key usage", lifecycle.KeyUsage)
	Style22.PrintL(w, "Rotation type", lifecycle.RotationType)
	Style22.Option(w, "Coordination ID", lifecycle.CoordinationID)
	Style22.PrintL(w, "Current credential", lifecycle.CurrentCredentialID)
	if lifecycle.NextCredentialID != nil {
		Style22.PrintL(w, "Next credential", *lifecycle.NextCredentialID)
	}
	if lifecycle.TriggerOperationID != nil {
		Style22.PrintL(w, "Trigger operation", *lifecycle.TriggerOperationID)
	}
	if lifecycle.LastOperationID != nil {
		Style22.PrintL(w, "Last operation", *lifecycle.LastOperationID)
	}
	if lifecycle.RequestedAt != nil {
		Style22.PrintL(w, "Requested at", *lifecycle.RequestedAt)
	}
	if lifecycle.SentAt != nil {
		Style22.PrintL(w, "Sent at", *lifecycle.SentAt)
	}
	if lifecycle.ActivatedAt != nil {
		Style22.PrintL(w, "Activated at", *lifecycle.ActivatedAt)
	}
	if lifecycle.RetiredAt != nil {
		Style22.PrintL(w, "Retired at", *lifecycle.RetiredAt)
	}
	Style22.Option(w, "Operator", lifecycle.Operator)
	Style22.Option(w, "Reason", lifecycle.Reason)
	if len(lifecycle.Evidence) > 0 {
		Style22.Printf(w, "Evidence:")
		displayMap(w, Style333, lifecycle.Evidence)
	}

	return nil
}

func displayEbicsKeyRotationGroup(w io.Writer, group *api.OutEbicsKeyRotationGroup) error {
	Style1.Printf(w, "EBICS key rotation %q", group.CoordinationID)
	if len(group.Operations) > 0 {
		Style22.PrintL(w, "Operations", len(group.Operations))
	}

	return displayEbicsKeyLifecycles(w, group.Lifecycles)
}

func displayEbicsInitializations(w io.Writer, workflows []*api.OutEbicsInitializationWorkflow) error {
	Style0.PrintV(w, "=== EBICS initializations ===")
	for _, workflow := range workflows {
		if err := displayEbicsInitialization(w, workflow); err != nil {
			return err
		}
	}

	return nil
}

func displayEbicsInitialization(w io.Writer, workflow *api.OutEbicsInitializationWorkflow) error {
	Style1.Printf(w, "EBICS initialization #%d [%s]", workflow.ID, workflow.Status)
	Style22.PrintL(w, "Current step", workflow.CurrentStep)
	if workflow.IniOperationID != nil {
		Style22.PrintL(w, "INI operation", *workflow.IniOperationID)
	}
	if workflow.HiaOperationID != nil {
		Style22.PrintL(w, "HIA operation", *workflow.HiaOperationID)
	}
	if workflow.H3KOperationID != nil {
		Style22.PrintL(w, "H3K operation", *workflow.H3KOperationID)
	}
	if workflow.LetterGeneratedAt != nil {
		Style22.PrintL(w, "Letter generated at", *workflow.LetterGeneratedAt)
	}
	if workflow.LetterConfirmedAt != nil {
		Style22.PrintL(w, "Letter confirmed at", *workflow.LetterConfirmedAt)
	}
	if workflow.BankActivationAt != nil {
		Style22.PrintL(w, "Bank activation at", *workflow.BankActivationAt)
	}
	Style22.Option(w, "Operator", workflow.Operator)
	Style22.Option(w, "Reason", workflow.Reason)
	Style22.Option(w, "Bank feedback", workflow.BankFeedback)
	if len(workflow.Evidence) > 0 {
		Style22.Printf(w, "Evidence:")
		displayMap(w, Style333, workflow.Evidence)
	}

	return nil
}

type EbicsKeyLifecycleList struct {
	ListOptions
}

func (c *EbicsKeyLifecycleList) Execute([]string) error { return execute(c) }
func (c *EbicsKeyLifecycleList) execute(w io.Writer) error {
	addr.Path = "/api/ebics/key-lifecycles"
	listURL(&c.ListOptions, "")

	var body map[string][]*api.OutEbicsKeyLifecycle
	if err := list(&body); err != nil {
		return err
	}

	if lifecycles := body["keyLifecycles"]; len(lifecycles) > 0 {
		return outputObject(w, lifecycles, &c.OutputFormat, displayEbicsKeyLifecycles)
	}

	fmt.Fprintln(w, "No EBICS key lifecycle found.")

	return nil
}

type EbicsKeyLifecycleGet struct {
	OutputFormat

	Args struct {
		Lifecycle string `required:"yes" positional-arg-name:"lifecycle" description:"The key lifecycle identifier"`
	} `positional-args:"yes"`
}

func (c *EbicsKeyLifecycleGet) Execute([]string) error { return execute(c) }
func (c *EbicsKeyLifecycleGet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/key-lifecycles", c.Args.Lifecycle)

	var lifecycle api.OutEbicsKeyLifecycle
	if err := get(&lifecycle); err != nil {
		return err
	}

	return outputObject(w, &lifecycle, &c.OutputFormat, displayEbicsKeyLifecycle)
}

type EbicsKeyLifecycleAction struct {
	Args struct {
		Lifecycle string `required:"yes" positional-arg-name:"lifecycle" description:"The key lifecycle identifier"`
	} `positional-args:"yes" json:"-"`

	Action   string             `required:"yes" long:"action" description:"The lifecycle action" json:"action,omitempty"`
	Operator string             `long:"operator" description:"The operator" json:"operator,omitempty"`
	Reason   string             `long:"reason" description:"The action reason" json:"reason,omitempty"`
	Evidence map[string]confVal `long:"evidence" description:"Structured evidence." json:"evidence,omitempty"`
}

func (c *EbicsKeyLifecycleAction) Execute([]string) error { return execute(c) }
func (c *EbicsKeyLifecycleAction) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/key-lifecycles", c.Args.Lifecycle, "actions")

	if err := updateMethod(w, c, "PUT"); err != nil {
		return err
	}

	fmt.Fprintf(w, "The EBICS key lifecycle %q was successfully updated.\n", c.Args.Lifecycle)

	return nil
}

//nolint:lll // CLI tags are intentionally explicit
type EbicsKeyRotationPrepare struct {
	OutputFormat

	EbicsSubscriberID              int64              `required:"yes" long:"subscriber" description:"The EBICS subscriber identifier"`
	CoordinationID                 string             `long:"coordination-id" description:"Reuse an existing coordinated rotation identifier"`
	RotationType                   string             `long:"rotation-type" choice:"ROTATION" choice:"REPLACEMENT" description:"The coordinated rotation type"`
	NextAuthenticationCredentialID *int64             `long:"auth-credential" description:"The next authentication credential identifier"`
	NextEncryptionCredentialID     *int64             `long:"enc-credential" description:"The next encryption credential identifier"`
	NextSignatureCredentialID      *int64             `long:"sig-credential" description:"The next signature credential identifier"`
	SignatureOrderType             string             `long:"signature-order-type" choice:"PUB" choice:"HSA" description:"The signature order to use for signature-only rotations"`
	Operator                       string             `long:"operator" description:"The operator"`
	Reason                         string             `long:"reason" description:"The action reason"`
	Evidence                       map[string]confVal `long:"evidence" description:"Structured evidence."`
}

func (c *EbicsKeyRotationPrepare) Execute([]string) error { return execute(c) }
func (c *EbicsKeyRotationPrepare) execute(w io.Writer) error {
	addr.Path = "/api/ebics/key-lifecycles/actions/prepare-rotation"

	req := api.InEbicsKeyRotationPrepare{
		EbicsSubscriberID:              c.EbicsSubscriberID,
		CoordinationID:                 c.CoordinationID,
		RotationType:                   c.RotationType,
		NextAuthenticationCredentialID: c.NextAuthenticationCredentialID,
		NextEncryptionCredentialID:     c.NextEncryptionCredentialID,
		NextSignatureCredentialID:      c.NextSignatureCredentialID,
		SignatureOrderType:             c.SignatureOrderType,
		Operator:                       c.Operator,
		Reason:                         c.Reason,
		Evidence:                       normalizeConfMap(c.Evidence),
	}

	var group api.OutEbicsKeyRotationGroup
	if err := postWorkflowAction(req, &group); err != nil {
		return err
	}

	return outputObject(w, &group, &c.OutputFormat, displayEbicsKeyRotationGroup)
}

//nolint:lll // CLI tags are intentionally explicit
type EbicsKeyRotationAction struct {
	OutputFormat

	Path               string             `json:"-" yaml:"-"`
	EbicsSubscriberID  int64              `required:"yes" long:"subscriber" description:"The EBICS subscriber identifier"`
	CoordinationID     string             `required:"yes" long:"coordination-id" description:"The coordinated rotation identifier"`
	SignatureOrderType string             `long:"signature-order-type" choice:"PUB" choice:"HSA" description:"The signature order to use for signature-only rotations"`
	SignatureDataPath  string             `long:"signature-data" description:"Path to the SPR signature data file"`
	Operator           string             `long:"operator" description:"The operator"`
	Reason             string             `long:"reason" description:"The action reason"`
	Evidence           map[string]confVal `long:"evidence" description:"Structured evidence."`
}

func (c *EbicsKeyRotationAction) Execute([]string) error { return execute(c) }
func (c *EbicsKeyRotationAction) execute(w io.Writer) error {
	addr.Path = c.Path

	req := api.InEbicsKeyRotationAction{
		EbicsSubscriberID:  c.EbicsSubscriberID,
		CoordinationID:     c.CoordinationID,
		SignatureOrderType: c.SignatureOrderType,
		Operator:           c.Operator,
		Reason:             c.Reason,
		Evidence:           normalizeConfMap(c.Evidence),
	}
	if c.SignatureDataPath != "" {
		data, err := os.ReadFile(c.SignatureDataPath)
		if err != nil {
			return fmt.Errorf("read coordinated rotation signature data %q: %w", c.SignatureDataPath, err)
		}
		req.SignatureData = data
	}

	var group api.OutEbicsKeyRotationGroup
	if err := postWorkflowAction(req, &group); err != nil {
		return err
	}

	return outputObject(w, &group, &c.OutputFormat, displayEbicsKeyRotationGroup)
}

type EbicsKeyRotationSend struct {
	EbicsKeyRotationAction
}

type EbicsKeyRotationConfirm struct {
	EbicsKeyRotationAction
}

type EbicsKeyRotationCancel struct {
	EbicsKeyRotationAction
}

type EbicsKeyRotationReject struct {
	EbicsKeyRotationAction
}

type EbicsKeyRotationRevoke struct {
	EbicsKeyRotationAction
}

func (c *EbicsKeyRotationSend) Execute(args []string) error {
	c.Path = "/api/ebics/key-lifecycles/actions/send-rotation"
	return c.EbicsKeyRotationAction.Execute(args)
}

func (c *EbicsKeyRotationConfirm) Execute(args []string) error {
	c.Path = "/api/ebics/key-lifecycles/actions/confirm-rotation"
	return c.EbicsKeyRotationAction.Execute(args)
}

func (c *EbicsKeyRotationCancel) Execute(args []string) error {
	c.Path = "/api/ebics/key-lifecycles/actions/cancel-rotation"
	return c.EbicsKeyRotationAction.Execute(args)
}

func (c *EbicsKeyRotationReject) Execute(args []string) error {
	c.Path = "/api/ebics/key-lifecycles/actions/reject-rotation"
	return c.EbicsKeyRotationAction.Execute(args)
}

func (c *EbicsKeyRotationRevoke) Execute(args []string) error {
	c.Path = "/api/ebics/key-lifecycles/actions/revoke-rotation"
	return c.EbicsKeyRotationAction.Execute(args)
}

func postWorkflowAction(request, out any) error {
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	resp, reqErr := sendRequest(ctx, request, http.MethodPost)
	if reqErr != nil {
		return reqErr
	}
	defer resp.Body.Close() //nolint:errcheck // nothing to handle

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
		return unmarshalBody(resp.Body, out)
	case http.StatusBadRequest:
		return getResponseErrorMessage(resp)
	case http.StatusNotFound:
		return getResponseErrorMessage(resp)
	default:
		return fmt.Errorf("unexpected response (%s): %w", resp.Status, getResponseErrorMessage(resp))
	}
}

type EbicsInitializationList struct {
	ListOptions
}

func (c *EbicsInitializationList) Execute([]string) error { return execute(c) }
func (c *EbicsInitializationList) execute(w io.Writer) error {
	addr.Path = "/api/ebics/initializations"
	listURL(&c.ListOptions, "")

	var body map[string][]*api.OutEbicsInitializationWorkflow
	if err := list(&body); err != nil {
		return err
	}

	if workflows := body["initializations"]; len(workflows) > 0 {
		return outputObject(w, workflows, &c.OutputFormat, displayEbicsInitializations)
	}

	fmt.Fprintln(w, "No EBICS initialization workflow found.")

	return nil
}

type EbicsInitializationGet struct {
	OutputFormat

	Args struct {
		Workflow string `required:"yes" positional-arg-name:"workflow" description:"The initialization workflow identifier"`
	} `positional-args:"yes"`
}

func (c *EbicsInitializationGet) Execute([]string) error { return execute(c) }
func (c *EbicsInitializationGet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/initializations", c.Args.Workflow)

	var workflow api.OutEbicsInitializationWorkflow
	if err := get(&workflow); err != nil {
		return err
	}

	return outputObject(w, &workflow, &c.OutputFormat, displayEbicsInitialization)
}

type EbicsInitializationAction struct {
	Args struct {
		Workflow string `required:"yes" positional-arg-name:"workflow" description:"The initialization workflow identifier"`
	} `positional-args:"yes" json:"-"`

	Action   string             `required:"yes" long:"action" description:"The init action" json:"action,omitempty"`
	Operator string             `long:"operator" description:"The operator" json:"operator,omitempty"`
	Reason   string             `long:"reason" description:"The action reason" json:"reason,omitempty"`
	Evidence map[string]confVal `long:"evidence" description:"Structured evidence." json:"evidence,omitempty"`
}

func (c *EbicsInitializationAction) Execute([]string) error { return execute(c) }
func (c *EbicsInitializationAction) execute(w io.Writer) error {
	addr.Path = path.Join("/api/ebics/initializations", c.Args.Workflow, "actions")

	if err := updateMethod(w, c, "PUT"); err != nil {
		return err
	}

	fmt.Fprintf(w, "The EBICS initialization workflow %q was successfully updated.\n", c.Args.Workflow)

	return nil
}
