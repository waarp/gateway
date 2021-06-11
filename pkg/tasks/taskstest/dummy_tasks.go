// Package taskstest defines a dummy transfer task which can be used for test
// purposes.
package taskstest

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

const (
	// ClientOK is a a dummy task type which can be used during client transfer
	// tests. The task always succeeds.
	ClientOK = "CLIENTOK"

	// ClientErr is a a dummy task type which can be used during client transfer
	// tests to check error handling. The task always fails.
	ClientErr = "CLIENTERR"

	// ServerOK is a a dummy task type which can be used during server transfer
	// tests. The task always succeeds.
	ServerOK = "SERVEROK"

	// ServerErr is a a dummy task type which can be used during server transfer
	// tests to check error handling. The task always fails.
	ServerErr = "SERVERERR"
)

var (
	// ClientCheckChannel is the channel used for checking the execution of the
	// client's dummy tasks during a transfer test.
	ClientCheckChannel chan string

	// ServerCheckChannel is the channel used for checking the execution of the
	// server's dummy tasks during a transfer test.
	ServerCheckChannel chan string
)

func init() {
	model.ValidTasks[ClientOK] = &testClientTask{}
	model.ValidTasks[ClientErr] = &testClientTaskError{}
	model.ValidTasks[ServerOK] = &testServerTask{}
	model.ValidTasks[ServerErr] = &testServerTaskError{}
}

// ##### CLIENT #####

type testClientTask struct{}

func (*testClientTask) Validate(map[string]string) error { return nil }
func (*testClientTask) Run(ctx context.Context, args map[string]string, _ *database.DB, c *model.TransferContext) (string, error) {

	msg := fmt.Sprintf("CLIENT | %s | %s | OK", c.Rule.Name, args["msg"])
	timer := time.NewTimer(time.Second)
	defer timer.Stop()

	select {
	case <-timer.C:
		panic(fmt.Sprintf("timeout while executing client task '%s'", msg))
	case ClientCheckChannel <- msg:
	case <-ctx.Done():
		return "", ctx.Err()
	}

	if d, ok := args["delay"]; ok {
		delay, err := strconv.ParseInt(d, 10, 64)
		if err != nil {
			return "", err
		}
		time.Sleep(time.Millisecond * time.Duration(delay))
	}
	return "", nil
}

type testClientTaskError struct{}

func (*testClientTaskError) Validate(map[string]string) error { return nil }
func (*testClientTaskError) Run(_ context.Context, args map[string]string, _ *database.DB, c *model.TransferContext) (string, error) {

	msg := fmt.Sprintf("CLIENT | %s | %s | ERROR", c.Rule.Name, args["msg"])
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-timer.C:
		panic(fmt.Sprintf("timeout while executing client task '%s'", msg))
	case ClientCheckChannel <- msg:
		return "task failed", fmt.Errorf("task failed")
	}
}

// ##### SERVER #####

type testServerTask struct{}

func (*testServerTask) Validate(map[string]string) error { return nil }
func (*testServerTask) Run(_ context.Context, args map[string]string, _ *database.DB, c *model.TransferContext) (string, error) {

	msg := fmt.Sprintf("SERVER | %s | %s | OK", c.Rule.Name, args["msg"])
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-timer.C:
		panic(fmt.Sprintf("timeout while executing server task '%s'", msg))
	case ServerCheckChannel <- msg:
		return "", nil
	}
}

type testServerTaskError struct{}

func (*testServerTaskError) Validate(map[string]string) error { return nil }
func (*testServerTaskError) Run(_ context.Context, args map[string]string, _ *database.DB, c *model.TransferContext) (string, error) {

	msg := fmt.Sprintf("SERVER | %s | %s | ERROR", c.Rule.Name, args["msg"])
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-timer.C:
		panic(fmt.Sprintf("timeout while executing server task '%s'", msg))
	case ServerCheckChannel <- msg:
		return "task failed", fmt.Errorf("task failed")
	}
}
