package wg

import (
	"fmt"
	"io"
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
	Style22.PrintL(w, "Current credential", lifecycle.CurrentCredentialID)
	if lifecycle.NextCredentialID != nil {
		Style22.PrintL(w, "Next credential", *lifecycle.NextCredentialID)
	}

	return nil
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
	Style22.Option(w, "Operator", workflow.Operator)
	Style22.Option(w, "Reason", workflow.Reason)
	Style22.Option(w, "Bank feedback", workflow.BankFeedback)

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
